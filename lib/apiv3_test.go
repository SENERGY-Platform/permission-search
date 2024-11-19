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
	"errors"
	"github.com/SENERGY-Platform/permission-search/lib/client"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"log"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
)

type TestAspect struct {
	model.EntryResult
	Name string `json:"name"`
}

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
	config.LogDeprecatedCallsToFile = ""
	config.FatalErrHandler = func(v ...interface{}) {
		log.Println("TEST-ERROR:", v)
		t.Log(v...)
	}

	config.OpenSearchInsecureSkipVerify = true
	config.OpenSearchUsername = "admin"
	config.OpenSearchPassword = "01J1iEnT#>kE"
	config.Debug = true

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

	c := client.NewClient("http://localhost:" + config.ServerPort)
	ctoken := testtoken

	t.Run("create aspects", createTestAspects(ctx, config, "aspect1", "aaaa", "aspect2", "aspect3", "aspect4", "aspect5"))

	time.Sleep(10 * time.Second) //kafka latency

	t.Run("export", func(t *testing.T) {
		list, err := c.ExportKind(admintoken, "aspects", 3, 1)
		if err != nil {
			t.Error(err)
			return
		}
		if len(list) != 3 {
			t.Error(len(list))
			return
		}
		expected := []model.ResourceRights{
			{
				ResourceRightsBase: model.ResourceRightsBase{
					UserRights: map[string]model.Right{"testOwner": {Read: true, Write: true, Execute: true, Administrate: true}},
					GroupRights: map[string]model.Right{
						"admin": {Read: true, Write: true, Execute: true, Administrate: true},
						"user":  {Read: true, Write: false, Execute: true, Administrate: false},
					},
				},
				ResourceId: "aspect1",
				Features:   map[string]interface{}{"name": "aspect1_name", "raw": map[string]interface{}{"name": "aspect1_name", "rdf_type": "aspect_type"}},
				Creator:    "testOwner",
			},
			{
				ResourceRightsBase: model.ResourceRightsBase{
					UserRights: map[string]model.Right{"testOwner": {Read: true, Write: true, Execute: true, Administrate: true}},
					GroupRights: map[string]model.Right{
						"admin": {Read: true, Write: true, Execute: true, Administrate: true},
						"user":  {Read: true, Write: false, Execute: true, Administrate: false},
					},
				},
				ResourceId: "aspect2",
				Features:   map[string]interface{}{"name": "aspect2_name", "raw": map[string]interface{}{"name": "aspect2_name", "rdf_type": "aspect_type"}},
				Creator:    "testOwner",
			},
			{
				ResourceRightsBase: model.ResourceRightsBase{
					UserRights: map[string]model.Right{"testOwner": {Read: true, Write: true, Execute: true, Administrate: true}},
					GroupRights: map[string]model.Right{
						"admin": {Read: true, Write: true, Execute: true, Administrate: true},
						"user":  {Read: true, Write: false, Execute: true, Administrate: false},
					},
				},
				ResourceId: "aspect3",
				Features:   map[string]interface{}{"name": "aspect3_name", "raw": map[string]interface{}{"name": "aspect3_name", "rdf_type": "aspect_type"}},
				Creator:    "testOwner",
			},
		}
		if !reflect.DeepEqual(expected, list) {
			t.Errorf("\ne:%#v\na:%#v\n", expected, list)
			return
		}
	})

	t.Run("client list", func(t *testing.T) {
		actual, err := c.List(ctoken, "aspects", client.ListOptions{})
		if err != nil {
			t.Error(err)
			return
		}
		expected := []map[string]interface{}{
			getTestAspectResult("aaaa"),
			getTestAspectResult("aspect1"),
			getTestAspectResult("aspect2"),
			getTestAspectResult("aspect3"),
			getTestAspectResult("aspect4"),
			getTestAspectResult("aspect5"),
		}
		actual = jsonNormaliseMap(actual)
		expected = jsonNormaliseMap(expected)
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("client list generic", func(t *testing.T) {
		actual, err := client.List[[]TestAspect](c, ctoken, "aspects", client.ListOptions{})
		if err != nil {
			t.Error(err)
			return
		}
		expected, err := client.JsonCast[[]TestAspect]([]map[string]interface{}{
			getTestAspectResult("aaaa"),
			getTestAspectResult("aspect1"),
			getTestAspectResult("aspect2"),
			getTestAspectResult("aspect3"),
			getTestAspectResult("aspect4"),
			getTestAspectResult("aspect5"),
		})
		if err != nil {
			t.Error(err)
			return
		}
		if len(expected) == 0 || expected[0].Name == "" {
			t.Error(expected)
			return
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("list", testRequest(config, "GET", "/v3/resources/aspects", nil, 200, []map[string]interface{}{
		getTestAspectResult("aaaa"),
		getTestAspectResult("aspect1"),
		getTestAspectResult("aspect2"),
		getTestAspectResult("aspect3"),
		getTestAspectResult("aspect4"),
		getTestAspectResult("aspect5"),
	}))

	t.Run("client list desc", func(t *testing.T) {
		actual, err := c.List(ctoken, "aspects", client.ListOptions{
			QueryListCommons: client.QueryListCommons{
				SortBy:   "name",
				SortDesc: true,
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected := []map[string]interface{}{
			getTestAspectResult("aspect5"),
			getTestAspectResult("aspect4"),
			getTestAspectResult("aspect3"),
			getTestAspectResult("aspect2"),
			getTestAspectResult("aspect1"),
			getTestAspectResult("aaaa"),
		}
		actual = jsonNormaliseMap(actual)
		expected = jsonNormaliseMap(expected)
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("client list generic desc", func(t *testing.T) {
		actual, err := client.List[[]TestAspect](c, ctoken, "aspects", client.ListOptions{
			QueryListCommons: client.QueryListCommons{
				SortBy:   "name",
				SortDesc: true,
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected, err := client.JsonCast[[]TestAspect]([]map[string]interface{}{
			getTestAspectResult("aspect5"),
			getTestAspectResult("aspect4"),
			getTestAspectResult("aspect3"),
			getTestAspectResult("aspect2"),
			getTestAspectResult("aspect1"),
			getTestAspectResult("aaaa"),
		})
		if err != nil {
			t.Error(err)
			return
		}
		if len(expected) == 0 || expected[0].Name == "" {
			t.Error(expected)
			return
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("list desc", testRequest(config, "GET", "/v3/resources/aspects?sort=name.desc", nil, 200, []map[string]interface{}{
		getTestAspectResult("aspect5"),
		getTestAspectResult("aspect4"),
		getTestAspectResult("aspect3"),
		getTestAspectResult("aspect2"),
		getTestAspectResult("aspect1"),
		getTestAspectResult("aaaa"),
	}))

	t.Run("client list limit offset", func(t *testing.T) {
		actual, err := c.List(ctoken, "aspects", client.ListOptions{
			QueryListCommons: client.QueryListCommons{
				Limit:  3,
				Offset: 1,
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected := []map[string]interface{}{
			getTestAspectResult("aspect1"),
			getTestAspectResult("aspect2"),
			getTestAspectResult("aspect3"),
		}
		actual = jsonNormaliseMap(actual)
		expected = jsonNormaliseMap(expected)
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("client list generic limit offset", func(t *testing.T) {
		actual, err := client.List[[]TestAspect](c, ctoken, "aspects", client.ListOptions{
			QueryListCommons: client.QueryListCommons{
				Limit:  3,
				Offset: 1,
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected, err := client.JsonCast[[]TestAspect]([]map[string]interface{}{
			getTestAspectResult("aspect1"),
			getTestAspectResult("aspect2"),
			getTestAspectResult("aspect3"),
		})
		if err != nil {
			t.Error(err)
			return
		}
		if len(expected) == 0 || expected[0].Name == "" {
			t.Error(expected)
			return
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("list limit offset", testRequest(config, "GET", "/v3/resources/aspects?limit=3&offset=1", nil, 200, []map[string]interface{}{
		getTestAspectResult("aspect1"),
		getTestAspectResult("aspect2"),
		getTestAspectResult("aspect3"),
	}))

	t.Run("client search aspect", func(t *testing.T) {
		actual, err := c.List(ctoken, "aspects", client.ListOptions{
			TextSearch: "aspect",
			QueryListCommons: client.QueryListCommons{
				Limit:  3,
				Offset: 1,
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected := []map[string]interface{}{
			getTestAspectResult("aspect2"),
			getTestAspectResult("aspect3"),
			getTestAspectResult("aspect4"),
		}
		actual = jsonNormaliseMap(actual)
		expected = jsonNormaliseMap(expected)
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("client search aspect generic", func(t *testing.T) {
		actual, err := client.List[[]TestAspect](c, ctoken, "aspects", client.ListOptions{
			TextSearch: "aspect",
			QueryListCommons: client.QueryListCommons{
				Limit:  3,
				Offset: 1,
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected, err := client.JsonCast[[]TestAspect]([]map[string]interface{}{
			getTestAspectResult("aspect2"),
			getTestAspectResult("aspect3"),
			getTestAspectResult("aspect4"),
		})
		if err != nil {
			t.Error(err)
			return
		}
		if len(expected) == 0 || expected[0].Name == "" {
			t.Error(expected)
			return
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("search aspect", testRequest(config, "GET", "/v3/resources/aspects?limit=3&offset=1&search=aspect", nil, 200, []map[string]interface{}{
		getTestAspectResult("aspect2"),
		getTestAspectResult("aspect3"),
		getTestAspectResult("aspect4"),
	}))

	t.Run("client search aaaa", func(t *testing.T) {
		actual, err := c.List(ctoken, "aspects", client.ListOptions{
			TextSearch: "aaaa",
			QueryListCommons: client.QueryListCommons{
				Limit: 3,
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected := []map[string]interface{}{
			getTestAspectResult("aaaa"),
		}
		actual = jsonNormaliseMap(actual)
		expected = jsonNormaliseMap(expected)
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("client search aaaa generic", func(t *testing.T) {
		actual, err := client.List[[]TestAspect](c, ctoken, "aspects", client.ListOptions{
			TextSearch: "aaaa",
			QueryListCommons: client.QueryListCommons{
				Limit: 3,
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected, err := client.JsonCast[[]TestAspect]([]map[string]interface{}{
			getTestAspectResult("aaaa"),
		})
		if err != nil {
			t.Error(err)
			return
		}
		if len(expected) == 0 || expected[0].Name == "" {
			t.Error(expected)
			return
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("search aaaa", testRequest(config, "GET", "/v3/resources/aspects?limit=3&search=aaaa", nil, 200, []map[string]interface{}{
		getTestAspectResult("aaaa"),
	}))

	t.Run("client ids aaaa", func(t *testing.T) {
		actual, err := c.List(ctoken, "aspects", client.ListOptions{
			QueryListCommons: client.QueryListCommons{
				Limit:  2,
				Offset: 1,
			},
			ListIds: []string{"aspect3", "aspect2", "aspect1"},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected := []map[string]interface{}{
			getTestAspectResult("aspect2"),
			getTestAspectResult("aspect3"),
		}
		actual = jsonNormaliseMap(actual)
		expected = jsonNormaliseMap(expected)
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("client ids generic", func(t *testing.T) {
		actual, err := client.List[[]TestAspect](c, ctoken, "aspects", client.ListOptions{
			QueryListCommons: client.QueryListCommons{
				Limit:  2,
				Offset: 1,
			},
			ListIds: []string{"aspect3", "aspect2", "aspect1"},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected, err := client.JsonCast[[]TestAspect]([]map[string]interface{}{
			getTestAspectResult("aspect2"),
			getTestAspectResult("aspect3"),
		})
		if err != nil {
			t.Error(err)
			return
		}
		if len(expected) == 0 || expected[0].Name == "" {
			t.Error(expected)
			return
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("ids", testRequest(config, "GET", "/v3/resources/aspects?limit=2&offset=1&ids=aspect3,aspect2,aspect1", nil, 200, []map[string]interface{}{
		getTestAspectResult("aspect2"),
		getTestAspectResult("aspect3"),
	}))

	t.Run("client filter", func(t *testing.T) {
		actual, err := c.List(ctoken, "aspects", client.ListOptions{
			Selection: &client.FeatureSelection{
				Feature: "name",
				Value:   "aspect4_name",
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected := []map[string]interface{}{
			getTestAspectResult("aspect4"),
		}
		actual = jsonNormaliseMap(actual)
		expected = jsonNormaliseMap(expected)
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("client filter generic", func(t *testing.T) {
		actual, err := client.List[[]TestAspect](c, ctoken, "aspects", client.ListOptions{
			Selection: &client.FeatureSelection{
				Feature: "name",
				Value:   "aspect4_name",
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected, err := client.JsonCast[[]TestAspect]([]map[string]interface{}{
			getTestAspectResult("aspect4"),
		})
		if err != nil {
			t.Error(err)
			return
		}
		if len(expected) == 0 || expected[0].Name == "" {
			t.Error(expected)
			return
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("filter", testRequest(config, "GET", "/v3/resources/aspects?limit=2&filter=name:aspect4_name", nil, 200, []map[string]interface{}{
		getTestAspectResult("aspect4"),
	}))

	t.Run("client access true", func(t *testing.T) {
		err := c.CheckUserOrGroup(ctoken, "aspects", "aspect5", "r")
		if err != nil {
			t.Errorf("expected no error, got %#v", err)
		}
	})

	t.Run("access true", testRequest(config, "GET", "/v3/resources/aspects/aspect5/access", nil, 200, true))

	t.Run("client access false", func(t *testing.T) {
		err := c.CheckUserOrGroup(ctoken, "aspects", "unknown", "r")
		if !errors.Is(err, client.ErrAccessDenied) {
			t.Errorf("expected ErrAccessDenied error, got: %#v", err)
		}
	})
	t.Run("access false", testRequest(config, "GET", "/v3/resources/aspects/unknown/access", nil, 200, false))

	t.Run("client query search", func(t *testing.T) {
		actual, _, err := c.Query(ctoken, client.QueryMessage{
			Resource: "aspects",
			Find: &client.QueryFind{
				QueryListCommons: client.QueryListCommons{
					Limit:    3,
					Offset:   1,
					SortBy:   "name",
					SortDesc: true,
				},
				Search: "aspect",
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected := jsonNormaliseValue([]map[string]interface{}{
			getTestAspectResult("aspect4"),
			getTestAspectResult("aspect3"),
			getTestAspectResult("aspect2"),
		})
		actual = jsonNormaliseValue(actual)
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("client query search generic", func(t *testing.T) {
		actual, _, err := client.Query[[]TestAspect](c, ctoken, client.QueryMessage{
			Resource: "aspects",
			Find: &client.QueryFind{
				QueryListCommons: client.QueryListCommons{
					Limit:    3,
					Offset:   1,
					SortBy:   "name",
					SortDesc: true,
				},
				Search: "aspect",
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected, err := client.JsonCast[[]TestAspect]([]map[string]interface{}{
			getTestAspectResult("aspect4"),
			getTestAspectResult("aspect3"),
			getTestAspectResult("aspect2"),
		})
		if err != nil {
			t.Error(err)
			return
		}
		if len(expected) == 0 || expected[0].Name == "" {
			t.Error(expected)
			return
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("client query search total", func(t *testing.T) {
		actual, _, err := client.QueryWithTotal[[]TestAspect](c, ctoken, client.QueryMessage{
			Resource: "aspects",
			Find: &client.QueryFind{
				QueryListCommons: client.QueryListCommons{
					Limit:    3,
					Offset:   1,
					SortBy:   "name",
					SortDesc: true,
				},
				Search: "aspect",
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		temp, err := client.JsonCast[[]TestAspect]([]map[string]interface{}{
			getTestAspectResult("aspect4"),
			getTestAspectResult("aspect3"),
			getTestAspectResult("aspect2"),
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected := client.WithTotal[[]TestAspect]{
			Result: temp,
			Total:  5,
		}
		if len(expected.Result) == 0 || expected.Result[0].Name == "" {
			t.Error(expected)
			return
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

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

	t.Run("client query search filter", func(t *testing.T) {
		actual, _, err := c.Query(ctoken, client.QueryMessage{
			Resource: "aspects",
			Find: &client.QueryFind{
				Filter: &client.Selection{
					Condition: client.ConditionConfig{
						Feature:   "features.name",
						Operation: "==",
						Value:     "aspect5_name",
					},
				},
				Search: "aspect",
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected := jsonNormaliseValue([]map[string]interface{}{
			getTestAspectResult("aspect5"),
		})
		actual = jsonNormaliseValue(actual)
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("client query search filter generic", func(t *testing.T) {
		actual, _, err := client.Query[[]TestAspect](c, ctoken, client.QueryMessage{
			Resource: "aspects",
			Find: &client.QueryFind{
				Filter: &client.Selection{
					Condition: client.ConditionConfig{
						Feature:   "features.name",
						Operation: "==",
						Value:     "aspect5_name",
					},
				},
				Search: "aspect",
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected, err := client.JsonCast[[]TestAspect]([]map[string]interface{}{
			getTestAspectResult("aspect5"),
		})
		if err != nil {
			t.Error(err)
			return
		}
		if len(expected) == 0 || expected[0].Name == "" {
			t.Error(expected)
			return
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("client query search filter with total", func(t *testing.T) {
		actual, _, err := client.QueryWithTotal[[]TestAspect](c, ctoken, client.QueryMessage{
			Resource: "aspects",
			Find: &client.QueryFind{
				Filter: &client.Selection{
					Condition: client.ConditionConfig{
						Feature:   "features.name",
						Operation: "==",
						Value:     "aspect5_name",
					},
				},
				Search: "aspect",
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		temp, err := client.JsonCast[[]TestAspect]([]map[string]interface{}{
			getTestAspectResult("aspect5"),
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected := client.WithTotal[[]TestAspect]{
			Result: temp,
			Total:  1,
		}
		if len(expected.Result) == 0 || expected.Result[0].Name == "" {
			t.Error(expected)
			return
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

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

	t.Run("client query filter", func(t *testing.T) {
		actual, _, err := c.Query(ctoken, client.QueryMessage{
			Resource: "aspects",
			Find: &client.QueryFind{
				Filter: &client.Selection{
					Condition: client.ConditionConfig{
						Feature:   "features.name",
						Operation: "==",
						Value:     "aspect5_name",
					},
				},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected := jsonNormaliseValue([]map[string]interface{}{
			getTestAspectResult("aspect5"),
		})
		actual = jsonNormaliseValue(actual)
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("client query filter generic", func(t *testing.T) {
		actual, _, err := client.Query[[]TestAspect](c, ctoken, client.QueryMessage{
			Resource: "aspects",
			Find: &client.QueryFind{
				Filter: &client.Selection{
					Condition: client.ConditionConfig{
						Feature:   "features.name",
						Operation: "==",
						Value:     "aspect5_name",
					},
				},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected, err := client.JsonCast[[]TestAspect]([]map[string]interface{}{
			getTestAspectResult("aspect5"),
		})
		if err != nil {
			t.Error(err)
			return
		}
		if len(expected) == 0 || expected[0].Name == "" {
			t.Error(expected)
			return
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("client query filter Total", func(t *testing.T) {
		actual, _, err := client.QueryWithTotal[[]TestAspect](c, ctoken, client.QueryMessage{
			Resource: "aspects",
			Find: &client.QueryFind{
				Filter: &client.Selection{
					Condition: client.ConditionConfig{
						Feature:   "features.name",
						Operation: "==",
						Value:     "aspect5_name",
					},
				},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		temp, err := client.JsonCast[[]TestAspect]([]map[string]interface{}{
			getTestAspectResult("aspect5"),
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected := client.WithTotal[[]TestAspect]{
			Result: temp,
			Total:  1,
		}
		if len(expected.Result) == 0 || expected.Result[0].Name == "" {
			t.Error(expected)
			return
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

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

	t.Run("query resource", testRequest(config, "POST", "/v3/query/aspects", model.QueryMessage{
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

	t.Run("query resource empty pl resource", testRequest(config, "POST", "/v3/query/aspects", model.QueryMessage{
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

	t.Run("client query ids 5", func(t *testing.T) {
		actual, _, err := c.Query(ctoken, client.QueryMessage{
			Resource: "aspects",
			ListIds: &client.QueryListIds{
				QueryListCommons: client.QueryListCommons{
					Limit:    3,
					Offset:   1,
					SortBy:   "name",
					SortDesc: true,
				},
				Ids: []string{"aspect1", "aspect2", "aspect3", "aspect4", "aspect5"},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected := jsonNormaliseValue([]map[string]interface{}{
			getTestAspectResult("aspect4"),
			getTestAspectResult("aspect3"),
			getTestAspectResult("aspect2"),
		})
		actual = jsonNormaliseValue(actual)
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("client query ids 5 generic", func(t *testing.T) {
		actual, _, err := client.Query[[]TestAspect](c, ctoken, client.QueryMessage{
			Resource: "aspects",
			ListIds: &client.QueryListIds{
				QueryListCommons: client.QueryListCommons{
					Limit:    3,
					Offset:   1,
					SortBy:   "name",
					SortDesc: true,
				},
				Ids: []string{"aspect1", "aspect2", "aspect3", "aspect4", "aspect5"},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected, err := client.JsonCast[[]TestAspect]([]map[string]interface{}{
			getTestAspectResult("aspect4"),
			getTestAspectResult("aspect3"),
			getTestAspectResult("aspect2"),
		})
		if err != nil {
			t.Error(err)
			return
		}
		if len(expected) == 0 || expected[0].Name == "" {
			t.Error(expected)
			return
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("client query ids 5 generic total", func(t *testing.T) {
		actual, _, err := client.QueryWithTotal[[]TestAspect](c, ctoken, client.QueryMessage{
			Resource: "aspects",
			ListIds: &client.QueryListIds{
				QueryListCommons: client.QueryListCommons{
					Limit:    3,
					Offset:   1,
					SortBy:   "name",
					SortDesc: true,
				},
				Ids: []string{"aspect1", "aspect2", "aspect3", "aspect4", "aspect5"},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		temp, err := client.JsonCast[[]TestAspect]([]map[string]interface{}{
			getTestAspectResult("aspect4"),
			getTestAspectResult("aspect3"),
			getTestAspectResult("aspect2"),
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected := client.WithTotal[[]TestAspect]{
			Result: temp,
			Total:  5,
		}
		if len(expected.Result) == 0 || expected.Result[0].Name == "" {
			t.Error(expected)
			return
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("query ids 5", testRequest(config, "POST", "/v3/query", model.QueryMessage{
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

	t.Run("client query ids 2", func(t *testing.T) {
		actual, _, err := c.Query(ctoken, client.QueryMessage{
			Resource: "aspects",
			ListIds: &client.QueryListIds{
				QueryListCommons: client.QueryListCommons{
					SortBy:   "name",
					SortDesc: true,
				},
				Ids: []string{"aspect2", "aspect3"},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected := jsonNormaliseValue([]map[string]interface{}{
			getTestAspectResult("aspect3"),
			getTestAspectResult("aspect2"),
		})
		actual = jsonNormaliseValue(actual)
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("client query ids 2 generic", func(t *testing.T) {
		actual, _, err := client.Query[[]TestAspect](c, ctoken, client.QueryMessage{
			Resource: "aspects",
			ListIds: &client.QueryListIds{
				QueryListCommons: client.QueryListCommons{
					SortBy:   "name",
					SortDesc: true,
				},
				Ids: []string{"aspect2", "aspect3"},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected, err := client.JsonCast[[]TestAspect]([]map[string]interface{}{
			getTestAspectResult("aspect3"),
			getTestAspectResult("aspect2"),
		})
		if err != nil {
			t.Error(err)
			return
		}
		if len(expected) == 0 || expected[0].Name == "" {
			t.Error(expected)
			return
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("query ids 2", testRequest(config, "POST", "/v3/query", model.QueryMessage{
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

	t.Run("client query check ids", func(t *testing.T) {
		actual, _, err := c.Query(ctoken, client.QueryMessage{
			Resource: "aspects",
			CheckIds: &client.QueryCheckIds{
				Ids: []string{"aspect2", "aspect3", "unknown"},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected := map[string]interface{}{
			"aspect2": true,
			"aspect3": true,
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("client query check ids generic", func(t *testing.T) {
		actual, _, err := client.Query[map[string]bool](c, ctoken, client.QueryMessage{
			Resource: "aspects",
			CheckIds: &client.QueryCheckIds{
				Ids: []string{"aspect2", "aspect3", "unknown"},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		expected := map[string]bool{
			"aspect2": true,
			"aspect3": true,
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

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
	expectedRightsModel := model.ResourceRights{}
	expectedRightsJson := `{"creator":"testOwner","features":{"name":"aspect2_name","raw":{"name":"aspect2_name","rdf_type":"aspect_type"}},"group_rights":{"admin":{"administrate":true,"execute":true,"read":true,"write":true},"user":{"administrate":false,"execute":true,"read":true,"write":false}},"resource_id":"aspect2","user_rights":{"testOwner":{"administrate":true,"execute":true,"read":true,"write":true}}}`
	json.Unmarshal([]byte(expectedRightsJson), &expectedRights)
	json.Unmarshal([]byte(expectedRightsJson), &expectedRightsModel)

	t.Run("client get aspect2 rights", func(t *testing.T) {
		actual, err := c.GetRights(ctoken, "aspects", "aspect2")
		if err != nil {
			t.Error(err)
			return
		}
		expected := expectedRightsModel
		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("\n%#v\n%#v\n", expected, actual)
			actualJson, _ := json.Marshal(actual)
			expectedJson, _ := json.Marshal(expected)
			t.Log("\n", string(actualJson), "\n", string(expectedJson))
		}
	})

	t.Run("get aspect2 rights", testRequest(config, "GET", "/v3/administrate/rights/aspects/aspect2", nil, 200, expectedRights))

	t.Run("client get rights of unknown", func(t *testing.T) {
		_, err := c.GetRights(ctoken, "aspects", "unknown")
		if !errors.Is(err, client.ErrAccessDenied) {
			t.Error(err)
			return
		}
	})

	t.Run("get rights of unknown", testRequest(config, "GET", "/v3/administrate/rights/aspects/unknown", nil, 403, nil))

	t.Run("client get rights of unknown with admin", func(t *testing.T) {
		_, err := c.GetRights(admintoken, "aspects", "unknown")
		if !errors.Is(err, client.ErrNotFound) {
			t.Error(err)
			return
		}
	})

	t.Run("get rights of unknown with admin", testRequestWithToken(config, admintoken, "GET", "/v3/administrate/rights/aspects/unknown", nil, 404, nil))

	t.Run("filter creator by jwt ref", testRequestWithToken(config, admintoken, "POST", "/v3/query/aspects", model.QueryMessage{
		Resource: "aspects",
		Find: &model.QueryFind{
			QueryListCommons: model.QueryListCommons{
				Rights:   "r",
				SortDesc: true,
			},
			Search: "",
			Filter: &model.Selection{
				Condition: model.ConditionConfig{
					Feature:   "creator",
					Operation: "!=",
					Ref:       "jwt.user",
				},
			},
		},
	}, 200, nil))

	t.Run("filter creator by jwt ref client", func(t *testing.T) {
		result, code, err := client.Query[[]any](c, admintoken, model.QueryMessage{
			Resource: "aspects",
			Find: &model.QueryFind{
				QueryListCommons: model.QueryListCommons{
					Rights:   "r",
					SortDesc: true,
				},
				Search: "",
				Filter: &model.Selection{
					Condition: model.ConditionConfig{
						Feature:   "creator",
						Operation: "!=",
						Ref:       "jwt.user",
					},
				},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		if code != 200 {
			t.Error(code)
			return
		}
		if len(result) == 0 {
			t.Error(result)
			return
		}

		result, code, err = client.Query[[]any](c, testtoken, model.QueryMessage{
			Resource: "aspects",
			Find: &model.QueryFind{
				QueryListCommons: model.QueryListCommons{
					Rights:   "r",
					SortDesc: true,
				},
				Search: "",
				Filter: &model.Selection{
					Condition: model.ConditionConfig{
						Feature:   "creator",
						Operation: "!=",
						Ref:       "jwt.user",
					},
				},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		if code != 200 {
			t.Error(code)
			return
		}
		if len(result) != 0 {
			t.Error(result)
			return
		}
	})

}

func jsonNormaliseMap(in []map[string]interface{}) (out []map[string]interface{}) {
	temp, _ := json.Marshal(in)
	json.Unmarshal(temp, &out)
	return
}

func jsonNormaliseValue(in interface{}) (out interface{}) {
	temp, _ := json.Marshal(in)
	json.Unmarshal(temp, &out)
	return
}
