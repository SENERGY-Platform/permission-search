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
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestApiV3(t *testing.T) {
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

	t.Run("create aspects", createTestAspects(ctx, config, "aspect1", "aaaa", "aspect2", "aspect3", "aspect4", "aspect5"))

	time.Sleep(10 * time.Second) //kafka latency

	t.Run("list", testRequest(config, "GET", "/v3/resources/aspects", nil, 200, []map[string]interface{}{
		getTestAspectResult("aaaa"),
		getTestAspectResult("aspect1"),
		getTestAspectResult("aspect2"),
		getTestAspectResult("aspect3"),
		getTestAspectResult("aspect4"),
		getTestAspectResult("aspect5"),
	}))

	t.Run("list desc", testRequest(config, "GET", "/v3/resources/aspects?sort=name.desc", nil, 200, []map[string]interface{}{
		getTestAspectResult("aspect5"),
		getTestAspectResult("aspect4"),
		getTestAspectResult("aspect3"),
		getTestAspectResult("aspect2"),
		getTestAspectResult("aspect1"),
		getTestAspectResult("aaaa"),
	}))

	t.Run("list limit offset", testRequest(config, "GET", "/v3/resources/aspects?limit=3&offset=1", nil, 200, []map[string]interface{}{
		getTestAspectResult("aspect1"),
		getTestAspectResult("aspect2"),
		getTestAspectResult("aspect3"),
	}))

	t.Run("search", testRequest(config, "GET", "/v3/resources/aspects?limit=3&offset=1&search=aspect", nil, 200, []map[string]interface{}{
		getTestAspectResult("aspect2"),
		getTestAspectResult("aspect3"),
		getTestAspectResult("aspect4"),
	}))

	t.Run("search", testRequest(config, "GET", "/v3/resources/aspects?limit=3&search=aaaa", nil, 200, []map[string]interface{}{
		getTestAspectResult("aaaa"),
	}))

	t.Run("ids", testRequest(config, "GET", "/v3/resources/aspects?limit=2&offset=1&ids=aspect3,aspect2,aspect1", nil, 200, []map[string]interface{}{
		getTestAspectResult("aspect2"),
		getTestAspectResult("aspect3"),
	}))

	t.Run("filter", testRequest(config, "GET", "/v3/resources/aspects?limit=2&filter=name:aspect4_name", nil, 200, []map[string]interface{}{
		getTestAspectResult("aspect4"),
	}))

	t.Run("access true", testRequest(config, "GET", "/v3/resources/aspects/aspect5/access", nil, 200, true))
	t.Run("access false", testRequest(config, "GET", "/v3/resources/aspects/unknown/access", nil, 200, false))

	t.Run("query search", testRequest(config, "POST", "/v3/query", model.QueryMessage{
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

	t.Run("query search filter", testRequest(config, "POST", "/v3/query", model.QueryMessage{
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

	t.Run("query filter", testRequest(config, "POST", "/v3/query", model.QueryMessage{
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

	t.Run("query ids", testRequest(config, "POST", "/v3/query", model.QueryMessage{
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

	t.Run("query ids", testRequest(config, "POST", "/v3/query", model.QueryMessage{
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

	t.Run("query check ids", testRequest(config, "POST", "/v3/query", model.QueryMessage{
		Resource: "aspects",
		CheckIds: &model.QueryCheckIds{
			Ids: []string{"aspect2", "aspect3", "unknown"},
		},
	}, 200, map[string]bool{
		"aspect2": true,
		"aspect3": true,
	}))

	expectedRights := map[string]interface{}{}
	json.Unmarshal([]byte(`{"creator":"testOwner","features":{"name":"aspect2_name","raw":{"name":"aspect2_name","rdf_type":"aspect_type"}},"group_rights":{"admin":{"administrate":true,"execute":true,"read":true,"write":true},"user":{"administrate":false,"execute":true,"read":true,"write":false}},"resource_id":"aspect2","user_rights":{"testOwner":{"administrate":true,"execute":true,"read":true,"write":true}}}`),
		&expectedRights)
	t.Run("get aspect2 rights", testRequest(config, "GET", "/v3/administrate/rights/aspects/aspect2", nil, 200, expectedRights))

	t.Run("get rights of unknown", testRequest(config, "GET", "/v3/administrate/rights/aspects/unknown", nil, 401, nil))

	t.Run("get rights of unknown", testRequest(config, "GET", "/v3/administrate/rights/aspects/unknown", nil, 401, nil))

	t.Run("get rights of unknown with admin", testRequestWithToken(config, admintoken, "GET", "/v3/administrate/rights/aspects/unknown", nil, 404, nil))

}
