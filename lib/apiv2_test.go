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
	"log"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestApiV2(t *testing.T) {
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
	config.OpenSearchPassword = "admin"

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

	t.Run("create aspects", createTestAspects(ctx, config, "aspect1", "aaaa", "aspect2", "aspect3", "aspect4", "aspect5"))

	time.Sleep(10 * time.Second) //kafka latency

	t.Run("list", testRequest(config, "GET", "/v2/aspects", nil, 200, []map[string]interface{}{
		getTestAspectResult("aaaa"),
		getTestAspectResult("aspect1"),
		getTestAspectResult("aspect2"),
		getTestAspectResult("aspect3"),
		getTestAspectResult("aspect4"),
		getTestAspectResult("aspect5"),
	}))

	t.Run("list desc", testRequest(config, "GET", "/v2/aspects?sort=name.desc", nil, 200, []map[string]interface{}{
		getTestAspectResult("aspect5"),
		getTestAspectResult("aspect4"),
		getTestAspectResult("aspect3"),
		getTestAspectResult("aspect2"),
		getTestAspectResult("aspect1"),
		getTestAspectResult("aaaa"),
	}))

	t.Run("list limit offset", testRequest(config, "GET", "/v2/aspects?limit=3&offset=1", nil, 200, []map[string]interface{}{
		getTestAspectResult("aspect1"),
		getTestAspectResult("aspect2"),
		getTestAspectResult("aspect3"),
	}))

	t.Run("search", testRequest(config, "GET", "/v2/aspects?limit=3&offset=1&search=aspect", nil, 200, []map[string]interface{}{
		getTestAspectResult("aspect2"),
		getTestAspectResult("aspect3"),
		getTestAspectResult("aspect4"),
	}))

	t.Run("search", testRequest(config, "GET", "/v2/aspects?limit=3&search=aaaa", nil, 200, []map[string]interface{}{
		getTestAspectResult("aaaa"),
	}))

	t.Run("ids", testRequest(config, "GET", "/v2/aspects?limit=2&offset=1&ids=aspect3,aspect2,aspect1", nil, 200, []map[string]interface{}{
		getTestAspectResult("aspect2"),
		getTestAspectResult("aspect3"),
	}))

	t.Run("filter", testRequest(config, "GET", "/v2/aspects?limit=2&filter=name:aspect4_name", nil, 200, []map[string]interface{}{
		getTestAspectResult("aspect4"),
	}))

	t.Run("access true", testRequest(config, "GET", "/v2/aspects/aspect5/access", nil, 200, true))
	t.Run("access false", testRequest(config, "GET", "/v2/aspects/unknown/access", nil, 200, false))

	t.Run("query search", testRequest(config, "POST", "/v2/query", model.QueryMessage{
		Resource: "aspects",
		Find: &model.QueryFind{
			QueryListCommons: model.QueryListCommons{
				Limit:    3,
				Offset:   1,
				SortBy:   "name",
				SortDesc: true,
			},
			Search: "aspect",
		},
	}, 200, []map[string]interface{}{
		getTestAspectResult("aspect4"),
		getTestAspectResult("aspect3"),
		getTestAspectResult("aspect2"),
	}))

	t.Run("query search filter", testRequest(config, "POST", "/v2/query", model.QueryMessage{
		Resource: "aspects",
		Find: &model.QueryFind{
			Filter: &model.Selection{
				Condition: model.ConditionConfig{
					Feature:   "features.name",
					Operation: "==",
					Value:     "aspect5_name",
				},
			},
			Search: "aspect",
		},
	}, 200, []map[string]interface{}{
		getTestAspectResult("aspect5"),
	}))

	t.Run("query filter", testRequest(config, "POST", "/v2/query", model.QueryMessage{
		Resource: "aspects",
		Find: &model.QueryFind{
			Filter: &model.Selection{
				Condition: model.ConditionConfig{
					Feature:   "features.name",
					Operation: "==",
					Value:     "aspect5_name",
				},
			},
		},
	}, 200, []map[string]interface{}{
		getTestAspectResult("aspect5"),
	}))

	t.Run("query ids", testRequest(config, "POST", "/v2/query", model.QueryMessage{
		Resource: "aspects",
		ListIds: &model.QueryListIds{
			QueryListCommons: model.QueryListCommons{
				Limit:    3,
				Offset:   1,
				SortBy:   "name",
				SortDesc: true,
			},
			Ids: []string{"aspect1", "aspect2", "aspect3", "aspect4", "aspect5"},
		},
	}, 200, []map[string]interface{}{
		getTestAspectResult("aspect4"),
		getTestAspectResult("aspect3"),
		getTestAspectResult("aspect2"),
	}))

	t.Run("query ids", testRequest(config, "POST", "/v2/query", model.QueryMessage{
		Resource: "aspects",
		ListIds: &model.QueryListIds{
			QueryListCommons: model.QueryListCommons{
				SortBy:   "name",
				SortDesc: true,
			},
			Ids: []string{"aspect2", "aspect3"},
		},
	}, 200, []map[string]interface{}{
		getTestAspectResult("aspect3"),
		getTestAspectResult("aspect2"),
	}))

	t.Run("query check ids", testRequest(config, "POST", "/v2/query", model.QueryMessage{
		Resource: "aspects",
		CheckIds: &model.QueryCheckIds{
			Ids: []string{"aspect2", "aspect3", "unknown"},
		},
	}, 200, map[string]bool{
		"aspect2": true,
		"aspect3": true,
	}))
}
