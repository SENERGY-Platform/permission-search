package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	k "github.com/SENERGY-Platform/permission-search/lib/worker/kafka"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestApiV2(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.LoadConfig("./../config.json")
	if err != nil {
		t.Error(err)
		return
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

func createTestAspects(ctx context.Context, config configuration.Config, ids ...string) func(t *testing.T) {
	return func(t *testing.T) {
		p, err := k.NewProducer(ctx, config.KafkaUrl, "aspects", true)
		if err != nil {
			t.Error(err)
			return
		}
		for _, id := range ids {
			t.Run("create "+id, createTestAspect(p, id))
		}
	}
}

func createTestAspect(p *k.Producer, id string) func(t *testing.T) {
	return func(t *testing.T) {
		aspectMsg, aspectCmd, err := getAspectTestObj(id, map[string]interface{}{
			"name":     id + "_name",
			"rdf_type": "aspect_type",
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

func testRequest(config configuration.Config, method string, path string, body interface{}, expectedStatusCode int, expected interface{}) func(t *testing.T) {
	return func(t *testing.T) {
		var requestBody io.Reader
		if body != nil {
			temp := new(bytes.Buffer)
			err := json.NewEncoder(temp).Encode(body)
			if err != nil {
				t.Error(err)
				return
			}
			requestBody = temp
		}

		req, err := http.NewRequest(method, "http://localhost:"+config.ServerPort+path, requestBody)
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
		if resp.StatusCode != expectedStatusCode {
			temp, _ := ioutil.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}

		temp, err := json.Marshal(expected)
		if err != nil {
			t.Error(err)
			return
		}
		var normalizedExpected interface{}
		err = json.Unmarshal(temp, &normalizedExpected)
		if err != nil {
			t.Error(err)
			return
		}

		var actual interface{}
		err = json.NewDecoder(resp.Body).Decode(&actual)
		if err != nil {
			t.Error(err)
			return
		}

		if !reflect.DeepEqual(actual, normalizedExpected) {
			a, _ := json.Marshal(actual)
			e, _ := json.Marshal(normalizedExpected)
			t.Error(string(a), string(e))
			return
		}
	}
}

func getTestAspectResult(id string) map[string]interface{} {
	//map[creator:testOwner id:aaspect name:aaspect_name permissions:map[a:true r:true w:true x:true] shared:false
	return map[string]interface{}{
		"creator": "testOwner",
		"id":      id,
		"name":    id + "_name",
		"permissions": map[string]bool{
			"a": true,
			"r": true,
			"w": true,
			"x": true,
		},
		"shared": false,
	}
}
