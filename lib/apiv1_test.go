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
	"encoding/json"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	k "github.com/SENERGY-Platform/permission-search/lib/worker/kafka"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestApiV1(t *testing.T) {
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
	config.LogDeprecatedCallsToFile = t.TempDir() + "/deprecated.log"
	config.FatalErrHandler = func(v ...interface{}) {
		log.Println("TEST-ERROR:", v)
		t.Log(v...)
	}

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
		zkUrl := zkIp + ":2181"

		//kafka
		config.KafkaUrl, err = Kafka(ctx, wg, zkUrl)
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

	t.Run("create aspect", func(t *testing.T) {
		aspectMsg, aspectCmd, err := getAspectTestObj("aspect1", map[string]interface{}{
			"name":     "aspect_name",
			"rdf_type": "aspect_type",
		})
		if err != nil {
			t.Error(err)
			return
		}
		p, err := k.NewProducer(ctx, config.KafkaUrl, "aspects", true)
		if err != nil {
			t.Error(err)
			return
		}
		err = p.Produce(aspectCmd.Id, aspectMsg)
		if err != nil {
			t.Error(err)
			return
		}
	})

	time.Sleep(10 * time.Second) //kafka latency

	t.Run("read aspect", func(t *testing.T) {
		//q.GetList(resource, "testOwner", []string{"user"}, "r", "3", "0", "name", true)
		path := "/jwt/list/aspects/r/3/0/name/asc"
		req, err := http.NewRequest("GET", "http://localhost:"+config.ServerPort+path, nil)
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
			temp, _ := ioutil.ReadAll(resp.Body)
			t.Error(resp.Status + ": " + string(temp))
			return
		}

		var result []map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			t.Error(err)
			return
		}

		if len(result) != 1 {
			t.Error(result)
			return
		}
		if !reflect.DeepEqual(result[0]["name"], "aspect_name") {
			t.Error(result)
			return
		}

	})

}
