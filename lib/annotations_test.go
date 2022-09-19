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
	"context"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	k "github.com/SENERGY-Platform/permission-search/lib/worker/kafka"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestAnnotations(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.LoadConfig("./../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	config.FatalErrHandler = t.Fatal
	config.Debug = true

	t.Run("start dependency containers", func(t *testing.T) {
		port, _, err := elasticsearch(ctx, wg)
		if err != nil {
			t.Error(err)
			return
		}
		config.ElasticUrl = "http://localhost:" + port

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

	t.Run("create devices", createTestDevices(ctx, config, "device1", "device2", "device3", "device4", "device5"))

	time.Sleep(20 * time.Second)
	t.Run("send connection state for device2 = connected", sendTestConnectionState(ctx, config, "device2", true))
	t.Run("send connection state for device3 = connected", sendTestConnectionState(ctx, config, "device3", true))
	t.Run("send connection state for device4 = disconnected", sendTestConnectionState(ctx, config, "device4", false))
	t.Run("send connection state for device2 = disconnected", sendTestConnectionState(ctx, config, "device2", false))

	time.Sleep(10 * time.Second) //kafka latency

	trueVar := true
	falseVar := false
	t.Run("query connected", testRequest(config, "POST", "/v3/query", model.QueryMessage{
		Resource: "devices",
		Find: &model.QueryFind{
			Filter: &model.Selection{
				Condition: model.ConditionConfig{
					Feature:   "annotations.connected",
					Operation: "==",
					Value:     true,
				},
			},
		},
	}, 200, []map[string]interface{}{
		getTestDeviceResult("device3", &trueVar),
	}))

	t.Run("query disconnected", testRequest(config, "POST", "/v3/query", model.QueryMessage{
		Resource: "devices",
		Find: &model.QueryFind{
			Filter: &model.Selection{
				Condition: model.ConditionConfig{
					Feature:   "annotations.connected",
					Operation: "==",
					Value:     false,
				},
			},
		},
	}, 200, []map[string]interface{}{
		getTestDeviceResult("device2", &falseVar),
		getTestDeviceResult("device4", &falseVar),
	}))

	t.Run("query unknown", testRequest(config, "POST", "/v3/query", model.QueryMessage{
		Resource: "devices",
		Find: &model.QueryFind{
			Filter: &model.Selection{
				Condition: model.ConditionConfig{
					Feature:   "annotations.connected",
					Operation: "==",
					Value:     nil,
				},
			},
		},
	}, 200, []map[string]interface{}{
		getTestDeviceResult("device1", nil),
		getTestDeviceResult("device5", nil),
	}))

}

func sendTestConnectionState(ctx context.Context, config configuration.Config, id string, connected bool) func(t *testing.T) {
	return func(t *testing.T) {
		p, err := k.NewProducer(ctx, config.KafkaUrl, "device_log", true)
		if err != nil {
			t.Error(err)
			return
		}
		err = p.Produce(id, []byte(`{"id": "`+id+`", "connected": `+strconv.FormatBool(connected)+`}`))
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func createTestDevices(ctx context.Context, config configuration.Config, ids ...string) func(t *testing.T) {
	return func(t *testing.T) {
		p, err := k.NewProducer(ctx, config.KafkaUrl, "devices", true)
		if err != nil {
			t.Error(err)
			return
		}
		for _, id := range ids {
			t.Run("create "+id, createTestDevice(p, id))
		}
	}
}

func createTestDevice(p *k.Producer, id string) func(t *testing.T) {
	return func(t *testing.T) {
		aspectMsg, aspectCmd, err := getDeviceTestObj(id, map[string]interface{}{
			"name": id + "_name",
		})
		if err != nil {
			t.Error(err)
			return
		}
		err = p.Produce(aspectCmd.Id, aspectMsg)
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func getTestDeviceResult(id string, connected *bool) (result map[string]interface{}) {
	result = map[string]interface{}{
		"creator":        "testOwner",
		"id":             id,
		"name":           id + "_name",
		"nickname":       nil,
		"display_name":   id + "_name",
		"attributes":     nil,
		"device_type_id": nil,
		"local_id":       nil,
		"permissions": map[string]bool{
			"a": true,
			"r": true,
			"w": true,
			"x": true,
		},
		"permission_holders": map[string][]string{
			"admin_users":   {"testOwner"},
			"execute_users": {"testOwner"},
			"read_users":    {"testOwner"},
			"write_users":   {"testOwner"},
		},
		"shared": false,
	}
	if connected != nil {
		result["annotations"] = map[string]interface{}{
			"connected": *connected,
		}
	}
	return result
}
