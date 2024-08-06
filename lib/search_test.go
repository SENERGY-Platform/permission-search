/*
 * Copyright 2022 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	k "github.com/SENERGY-Platform/permission-search/lib/worker/kafka"
	"github.com/google/uuid"
	"io"
	"log"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"
)

func BenchmarkSearch(b *testing.B) {
	b.Skip("benchmark")
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.LoadConfig("./../config.json")
	if err != nil {
		b.Error(err)
		return
	}
	config.LogDeprecatedCallsToFile = ""
	config.FatalErrHandler = func(v ...interface{}) {
		log.Println("TEST-ERROR:", v)
		b.Log(v...)
	}

	config.OpenSearchInsecureSkipVerify = true
	config.OpenSearchUsername = "admin"
	config.OpenSearchPassword = "01J1iEnT#>kE"

	b.Run("start dependency containers", func(b *testing.B) {
		_, ip, err := OpenSearch(ctx, wg)
		if err != nil {
			b.Error(err)
			return
		}
		config.OpenSearchUrls = "https://" + ip + ":9200"

		_, zkIp, err := Zookeeper(ctx, wg)
		if err != nil {
			b.Error(err)
			return
		}
		config.KafkaUrl = zkIp + ":2181"

		//kafka
		config.KafkaUrl, err = Kafka(ctx, wg, config.KafkaUrl)
		if err != nil {
			b.Error(err)
			return
		}
	})

	b.Run("start server", func(b *testing.B) {
		freePort, err := GetFreePort()
		if err != nil {
			b.Error(err)
			return
		}
		config.ServerPort = strconv.Itoa(freePort)
		err = Start(ctx, config, Standalone)
		if err != nil {
			b.Error(err)
			return
		}
	})

	deviceNames := []string{"Plug Kühlschrank Backofen ", "HEAT_COST_ALLOCATOR", "HEAT-COST-ALLOCATOR", "HEAT COST ALLOCATOR", "HeatCostAllocator", "foo", "cator", "heal", "heat"}
	/*
		for i := len(deviceNames); i < 1000; i++ {
			deviceNames = append(deviceNames, uuid.NewString())
		}
	*/

	b.Run("create devices", createSearchTestDevicesForBenchmark(ctx, config, deviceNames...))

	time.Sleep(10 * time.Second) //kafka latency

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		b.Run("check cost", checkDeviceSearchForBenchmark(config, "cost"))
		b.Run("check COST", checkDeviceSearchForBenchmark(config, "COST"))
		b.Run("check HEAT", checkDeviceSearchForBenchmark(config, "HEAT"))
		b.Run("check heat", checkDeviceSearchForBenchmark(config, "heat"))

		b.Run("check HEAT_COST", checkDeviceSearchForBenchmark(config, "HEAT_COST"))
		b.Run("check HeatCost", checkDeviceSearchForBenchmark(config, "HeatCost"))
		b.Run("check COST_ALLOCATOR", checkDeviceSearchForBenchmark(config, "COST_ALLOCATOR"))
		b.Run("check CostAllocator", checkDeviceSearchForBenchmark(config, "CostAllocator"))

		b.Run("check HEAT-COST", checkDeviceSearchForBenchmark(config, "HEAT-COST"))
		b.Run("check Heat-Cost", checkDeviceSearchForBenchmark(config, "Heat-Cost"))
		b.Run("check COST-ALLOCATOR", checkDeviceSearchForBenchmark(config, "COST-ALLOCATOR"))
		b.Run("check Cost-Allocator", checkDeviceSearchForBenchmark(config, "Cost-Allocator"))

		b.Run("check Allo", checkDeviceSearchForBenchmark(config, "Allo"))

		b.Run("check Hea", checkDeviceSearchForBenchmark(config, "Hea"))

		b.Run("check küh", checkDeviceSearchForBenchmark(config, "küh"))
		b.Run("check back", checkDeviceSearchForBenchmark(config, "back"))

		b.Run("check CATOR", checkDeviceSearchForBenchmark(config, "CATOR"))
		b.Run("check cator", checkDeviceSearchForBenchmark(config, "cator"))
	}

}

func TestSearch(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.LoadConfig("./../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	config.LogDeprecatedCallsToFile = ""
	config.FatalErrHandler = func(v ...interface{}) {
		log.Println("TEST-ERROR:", v)
		t.Log(v...)
	}

	config.OpenSearchInsecureSkipVerify = true
	config.OpenSearchUsername = "admin"
	config.OpenSearchPassword = "01J1iEnT#>kE"

	t.Run("start dependency containers", func(t *testing.T) {
		_, ip, err := OpenSearch(ctx, wg)
		if err != nil {
			t.Error(err)
			return
		}
		config.OpenSearchUrls = "https://" + ip + ":9200"

		_, zkIp, err := Zookeeper(ctx, wg)
		if err != nil {
			t.Error(err)
			return
		}
		config.KafkaUrl = zkIp + ":2181"

		//kafka
		config.KafkaUrl, err = Kafka(ctx, wg, config.KafkaUrl)
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("start server", func(t *testing.T) {
		freePort, err := GetFreePort()
		if err != nil {
			t.Error(err)
			return
		}
		config.ServerPort = strconv.Itoa(freePort)
		err = Start(ctx, config, Standalone)
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("create devices", createSearchTestDevices(ctx, config, "Plug Kühlschrank Backofen ", "HEAT_COST_ALLOCATOR", "HEAT-COST-ALLOCATOR", "HEAT COST ALLOCATOR", "HeatCostAllocator", "foo", "cator", "heal", "heat"))

	time.Sleep(10 * time.Second) //kafka latency

	t.Run("check cost", checkDeviceSearch(config, "cost", "HEAT_COST_ALLOCATOR", "HEAT-COST-ALLOCATOR", "HEAT COST ALLOCATOR", "HeatCostAllocator"))
	t.Run("check COST", checkDeviceSearch(config, "COST", "HEAT_COST_ALLOCATOR", "HEAT-COST-ALLOCATOR", "HEAT COST ALLOCATOR", "HeatCostAllocator"))
	t.Run("check HEAT", checkDeviceSearch(config, "HEAT", "HEAT_COST_ALLOCATOR", "HEAT-COST-ALLOCATOR", "HEAT COST ALLOCATOR", "HeatCostAllocator", "heat"))
	t.Run("check heat", checkDeviceSearch(config, "heat", "HEAT_COST_ALLOCATOR", "HEAT-COST-ALLOCATOR", "HEAT COST ALLOCATOR", "HeatCostAllocator", "heat"))

	t.Run("check HEAT_COST", checkDeviceSearch(config, "HEAT_COST", "HEAT_COST_ALLOCATOR", "HEAT-COST-ALLOCATOR", "HEAT COST ALLOCATOR", "HeatCostAllocator"))
	t.Run("check HeatCost", checkDeviceSearch(config, "HeatCost", "HEAT_COST_ALLOCATOR", "HEAT-COST-ALLOCATOR", "HEAT COST ALLOCATOR", "HeatCostAllocator"))
	t.Run("check COST_ALLOCATOR", checkDeviceSearch(config, "COST_ALLOCATOR", "HEAT_COST_ALLOCATOR", "HEAT-COST-ALLOCATOR", "HEAT COST ALLOCATOR", "HeatCostAllocator"))
	t.Run("check CostAllocator", checkDeviceSearch(config, "CostAllocator", "HEAT_COST_ALLOCATOR", "HEAT-COST-ALLOCATOR", "HEAT COST ALLOCATOR", "HeatCostAllocator"))

	t.Run("check HEAT-COST", checkDeviceSearch(config, "HEAT-COST", "HEAT_COST_ALLOCATOR", "HEAT-COST-ALLOCATOR", "HEAT COST ALLOCATOR", "HeatCostAllocator"))
	t.Run("check Heat-Cost", checkDeviceSearch(config, "Heat-Cost", "HEAT_COST_ALLOCATOR", "HEAT-COST-ALLOCATOR", "HEAT COST ALLOCATOR", "HeatCostAllocator"))
	t.Run("check COST-ALLOCATOR", checkDeviceSearch(config, "COST-ALLOCATOR", "HEAT_COST_ALLOCATOR", "HEAT-COST-ALLOCATOR", "HEAT COST ALLOCATOR", "HeatCostAllocator"))
	t.Run("check Cost-Allocator", checkDeviceSearch(config, "Cost-Allocator", "HEAT_COST_ALLOCATOR", "HEAT-COST-ALLOCATOR", "HEAT COST ALLOCATOR", "HeatCostAllocator"))

	t.Run("check Allo", checkDeviceSearch(config, "Allo", "HEAT_COST_ALLOCATOR", "HEAT-COST-ALLOCATOR", "HEAT COST ALLOCATOR", "HeatCostAllocator"))

	t.Run("check Hea", checkDeviceSearch(config, "Hea", "HEAT_COST_ALLOCATOR", "HEAT-COST-ALLOCATOR", "HEAT COST ALLOCATOR", "HeatCostAllocator", "heal", "heat"))

	t.Run("check küh", checkDeviceSearch(config, "küh", "Plug Kühlschrank Backofen "))
	t.Run("check back", checkDeviceSearch(config, "back", "Plug Kühlschrank Backofen "))

	t.Run("check CATOR", checkDeviceSearch(config, "CATOR", "HEAT_COST_ALLOCATOR", "HEAT-COST-ALLOCATOR", "HEAT COST ALLOCATOR", "HeatCostAllocator", "cator"))
	t.Run("check cator", checkDeviceSearch(config, "cator", "HEAT_COST_ALLOCATOR", "HEAT-COST-ALLOCATOR", "HEAT COST ALLOCATOR", "HeatCostAllocator", "cator"))
}

func checkDeviceSearch(config configuration.Config, searchText string, expectedResultNames ...string) func(t *testing.T) {
	return func(t *testing.T) {
		method := "POST"
		path := "/v3/query"
		body := new(bytes.Buffer)
		err := json.NewEncoder(body).Encode(model.QueryMessage{
			Resource: "devices",
			Find: &model.QueryFind{
				QueryListCommons: model.QueryListCommons{
					Limit:    100,
					Offset:   0,
					SortBy:   "name",
					SortDesc: true,
				},
				Search: searchText,
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		req, err := http.NewRequest(method, "http://localhost:"+config.ServerPort+path, body)
		if err != nil {
			t.Error(err)
			return
		}
		req.Header.Set("Authorization", testtoken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
			return
		}
		if resp.StatusCode != 200 {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}
		devices := []NameWrapper{}
		err = json.NewDecoder(resp.Body).Decode(&devices)
		if err != nil {
			t.Error(err)
			return
		}
		actualNames := []string{}
		for _, device := range devices {
			actualNames = append(actualNames, device.Name)
		}
		sort.Strings(actualNames)
		expectedNames := append([]string{}, expectedResultNames...)
		sort.Strings(expectedNames)
		if !reflect.DeepEqual(expectedNames, actualNames) {
			t.Errorf("\n%#v\n%#v\n", actualNames, expectedNames)
		}
	}
}

func checkDeviceSearchForBenchmark(config configuration.Config, searchText string) func(b *testing.B) {
	return func(b *testing.B) {
		method := "POST"
		path := "/v3/query"
		body := new(bytes.Buffer)
		err := json.NewEncoder(body).Encode(model.QueryMessage{
			Resource: "devices",
			Find: &model.QueryFind{
				QueryListCommons: model.QueryListCommons{
					Limit:    100,
					Offset:   0,
					SortBy:   "name",
					SortDesc: true,
				},
				Search: searchText,
			},
		})
		if err != nil {
			b.Error(err)
			return
		}
		req, err := http.NewRequest(method, "http://localhost:"+config.ServerPort+path, body)
		if err != nil {
			b.Error(err)
			return
		}
		req.Header.Set("Authorization", testtoken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			b.Error(err)
			return
		}
		if resp.StatusCode != 200 {
			temp, _ := io.ReadAll(resp.Body)
			b.Error(resp.StatusCode, string(temp))
			return
		}
	}
}

type NameWrapper struct {
	Name string `json:"name"`
}

func createSearchTestDevices(ctx context.Context, config configuration.Config, names ...string) func(t *testing.T) {
	return func(t *testing.T) {
		p, err := k.NewProducer(ctx, config.KafkaUrl, "devices", true)
		if err != nil {
			t.Error(err)
			return
		}
		for _, name := range names {
			t.Run("create "+name, createSearchTestDevice(p, name))
		}
	}
}

func createSearchTestDevicesForBenchmark(ctx context.Context, config configuration.Config, names ...string) func(b *testing.B) {
	return func(b *testing.B) {
		p, err := k.NewProducer(ctx, config.KafkaUrl, "devices", true)
		if err != nil {
			b.Error(err)
			return
		}
		for _, name := range names {
			b.Run("create "+name, createSearchTestDeviceForBenchmark(p, name))
		}
	}
}

func createSearchTestDevice(p *k.Producer, name string) func(t *testing.T) {
	return func(t *testing.T) {
		deviceMsg, deviceCmd, err := getDeviceTestObj(uuid.New().String(), map[string]interface{}{
			"name": name,
		})
		if err != nil {
			t.Error(err)
			return
		}
		err = p.Produce(deviceCmd.Id, deviceMsg)
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func createSearchTestDeviceForBenchmark(p *k.Producer, name string) func(b *testing.B) {
	return func(b *testing.B) {
		deviceMsg, deviceCmd, err := getDeviceTestObj(uuid.New().String(), map[string]interface{}{
			"name": name,
		})
		if err != nil {
			b.Error(err)
			return
		}
		err = p.Produce(deviceCmd.Id, deviceMsg)
		if err != nil {
			b.Error(err)
			return
		}
	}
}
