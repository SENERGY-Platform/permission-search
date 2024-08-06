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
	"fmt"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/worker"
	"log"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestSearchAfter(t *testing.T) {
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

	var w *worker.Worker
	t.Run("start server", func(t *testing.T) {
		freePort, err := GetFreePort()
		if err != nil {
			t.Error(err)
			return
		}
		config.ServerPort = strconv.Itoa(freePort)
		_, _, w, err = StartGetComponents(ctx, config, Standalone)
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("create entries", func(t *testing.T) {
		handler := w.GetResourceCommandHandler("aspects")
		for i := 1; i <= 20000; i++ {
			id := fmt.Sprintf("%06d", i)
			log.Println("create", id)
			aspectMsg, _, err := getAspectTestObj(id, map[string]interface{}{
				"name":     id + "_name",
				"rdf_type": "aspect_type",
			})
			if err != nil {
				t.Error(err)
				return
			}
			err = handler(aspectMsg)
			if err != nil {
				t.Error(err)
				return
			}
		}
	})

	time.Sleep(1 * time.Second)

	t.Run("access true --> last entry exists", testRequest(config, "GET", "/v3/resources/aspects/020000/access", nil, 200, true))

	t.Run("list offset 9900", testRequest(config, "GET", "/v3/resources/aspects?limit=3&offset=9900", nil, 200, []map[string]interface{}{
		getTestAspectResult("009901"),
		getTestAspectResult("009902"),
		getTestAspectResult("009903"),
	}))

	t.Run("list after 009900", testRequest(config, "GET", "/v3/resources/aspects?limit=3&sort=id&after.id=009900", nil, 200, []map[string]interface{}{
		getTestAspectResult("009901"),
		getTestAspectResult("009902"),
		getTestAspectResult("009903"),
	}))
	t.Run("list after 009900", testRequest(config, "GET", "/v3/resources/aspects?limit=3&sort=resource&after.id=009900", nil, 200, []map[string]interface{}{
		getTestAspectResult("009901"),
		getTestAspectResult("009902"),
		getTestAspectResult("009903"),
	}))

	t.Run("list after 009900 desc", testRequest(config, "GET", "/v3/resources/aspects?limit=3&sort=id.desc&after.id=009900", nil, 200, []map[string]interface{}{
		getTestAspectResult("009899"),
		getTestAspectResult("009898"),
		getTestAspectResult("009897"),
	}))
	t.Run("list after 009900 desc", testRequest(config, "GET", "/v3/resources/aspects?limit=3&sort=resource.desc&after.id=009900", nil, 200, []map[string]interface{}{
		getTestAspectResult("009899"),
		getTestAspectResult("009898"),
		getTestAspectResult("009897"),
	}))

	t.Run("list offset 19900", testRequest(config, "GET", "/v3/resources/aspects?limit=3&sort=id&offset=19900", nil, http.StatusBadRequest, nil))

	t.Run("list offset 19900", testRequest(config, "GET", "/v3/resources/aspects?limit=3&sort=resource&offset=19900", nil, http.StatusBadRequest, nil))

	t.Run("list after 019900", testRequest(config, "GET", "/v3/resources/aspects?limit=3&sort=id&after.id=019900", nil, 200, []map[string]interface{}{
		getTestAspectResult("019901"),
		getTestAspectResult("019902"),
		getTestAspectResult("019903"),
	}))

	t.Run("list after 019900", testRequest(config, "GET", "/v3/resources/aspects?limit=3&sort=resource&after.id=019900", nil, 200, []map[string]interface{}{
		getTestAspectResult("019901"),
		getTestAspectResult("019902"),
		getTestAspectResult("019903"),
	}))

	t.Run("list after 019900 desc", testRequest(config, "GET", "/v3/resources/aspects?limit=3&sort=id.desc&after.id=019900", nil, 200, []map[string]interface{}{
		getTestAspectResult("019899"),
		getTestAspectResult("019898"),
		getTestAspectResult("019897"),
	}))

	t.Run("list after 019900 desc", testRequest(config, "GET", "/v3/resources/aspects?limit=3&sort=resource.desc&after.id=019900", nil, 200, []map[string]interface{}{
		getTestAspectResult("019899"),
		getTestAspectResult("019898"),
		getTestAspectResult("019897"),
	}))
}
