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
	"crypto/tls"
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/opensearchclient"
	k "github.com/SENERGY-Platform/permission-search/lib/worker/kafka"
	"github.com/opensearch-project/opensearch-go"
	"github.com/opensearch-project/opensearch-go/opensearchapi"
	"github.com/opensearch-project/opensearch-go/opensearchutil"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMappingUpdateForNewField(t *testing.T) {
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
	config.Debug = true

	config.OpenSearchInsecureSkipVerify = true
	config.OpenSearchUsername = "admin"
	config.OpenSearchPassword = "01J1iEnT#>kE"

	config.TryMappingUpdateOnStartup = true

	t.Run("start dependency containers", func(t *testing.T) {
		_, ip, err := OpenSearch(ctx, wg)
		if err != nil {
			t.Error(err)
			return
		}
		config.OpenSearchUrls = "https://" + ip + ":9200"
	})

	config.Resources = map[string]configuration.ResourceConfig{"test_resource_kind": {}}
	config.ResourceList = []string{"test_resource_kind"}
	config.IndexTypeMapping = map[string]map[string]map[string]interface{}{}

	var client *opensearch.Client

	t.Run("init", func(t *testing.T) {
		config.IndexTypeMapping["test_resource_kind"] = map[string]map[string]interface{}{
			"features": {
				"stable":  map[string]interface{}{"type": "keyword"},
				"removed": map[string]interface{}{"type": "keyword"},
				"changed": map[string]interface{}{"type": "keyword"},
			},
		}
		client, err = opensearchclient.New(config)
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("add doc after init", func(t *testing.T) {
		options := []func(request *opensearchapi.IndexRequest){
			client.Index.WithDocumentID("init"),
			client.Index.WithContext(ctx),
		}
		resp, err := client.Index(
			"test_resource_kind",
			opensearchutil.NewJSONReader(map[string]interface{}{
				"stable":  "stable",
				"removed": "removed",
				"changed": "changed",
			}),
			options...,
		)
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		if resp.IsError() {
			t.Error(errors.New(resp.String()))
			return
		}
	})

	t.Run("add and remove", func(t *testing.T) {
		config.IndexTypeMapping["test_resource_kind"] = map[string]map[string]interface{}{
			"features": {
				"stable":  map[string]interface{}{"type": "keyword"},
				"added":   map[string]interface{}{"type": "keyword"},
				"changed": map[string]interface{}{"type": "keyword"},
			},
		}
		_, err = opensearchclient.New(config)
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("add doc after add/remove", func(t *testing.T) {
		options := []func(request *opensearchapi.IndexRequest){
			client.Index.WithDocumentID("add_remove"),
			client.Index.WithContext(ctx),
		}
		resp, err := client.Index(
			"test_resource_kind",
			opensearchutil.NewJSONReader(map[string]interface{}{
				"stable":  "stable",
				"added":   "added",
				"changed": "changed",
			}),
			options...,
		)
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		if resp.IsError() {
			t.Error(errors.New(resp.String()))
			return
		}
	})

	t.Run("change", func(t *testing.T) {
		t.Log("illegal change --> error will be logged but ignored")
		config.IndexTypeMapping["test_resource_kind"] = map[string]map[string]interface{}{
			"features": {
				"stable":  map[string]interface{}{"type": "keyword"},
				"added":   map[string]interface{}{"type": "keyword"},
				"changed": map[string]interface{}{"type": "boolean"},
			},
		}
		_, err = opensearchclient.New(config)
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("add doc after change", func(t *testing.T) {
		options := []func(request *opensearchapi.IndexRequest){
			client.Index.WithDocumentID("change"),
			client.Index.WithContext(ctx),
		}
		resp, err := client.Index(
			"test_resource_kind",
			opensearchutil.NewJSONReader(map[string]interface{}{
				"stable":  "stable",
				"added":   "added",
				"changed": true,
			}),
			options...,
		)
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		if resp.IsError() {
			t.Error(errors.New(resp.String()))
			return
		}
	})

}

func TestMappingUpdate(t *testing.T) {
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
	config.Debug = true

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

	t.Run("start server", startTestServer(config, confV1, ctx))

	t.Run("add elements", testMappingAddElements(config))

	time.Sleep(5 * time.Second)

	t.Run("read by name v1", testMappingReadBy(config, "name", "n1", 1))
	//t.Run("read by classId v1", testMappingReadBy(config, "device_class_id", "dc1", 0))
	t.Run("search dc1 v1", testMappingSearch(config, "dc1", 0))
	t.Run("search n1 v1", testMappingSearch(config, "n1", 1))
	t.Run("check index version v1", testCheckIndexVersion(config, "device-types", "device-types_v1"))

	t.Run("update index", testUpdateIndex(config, confV2))
	time.Sleep(5 * time.Second)
	t.Run("read by name v2", testMappingReadBy(config, "name", "n1", 1))
	t.Run("read by classId v2", testMappingReadBy(config, "device_class_id", "dc1", 1))
	t.Run("search dc1 v2", testMappingSearch(config, "dc1", 1))
	t.Run("search n1 v2", testMappingSearch(config, "n1", 1))
	t.Run("check index version v2", testCheckIndexVersion(config, "device-types", "device-types_v2"))
}

func testCheckIndexVersion(config configuration.Config, kind string, expectedCurrent string) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		client, err := opensearch.NewClient(opensearch.Config{
			EnableRetryOnTimeout:  true,
			MaxRetries:            config.MaxRetry,
			RetryBackoff:          func(i int) time.Duration { return time.Duration(i) * 100 * time.Millisecond },
			DiscoverNodesOnStart:  true,
			DiscoverNodesInterval: time.Minute,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: config.OpenSearchInsecureSkipVerify},
			},
			Addresses: strings.Split(config.OpenSearchUrls, ","),
			Username:  config.OpenSearchUsername, // For testing only. Don't store credentials in code.
			Password:  config.OpenSearchPassword,
		})
		if err != nil {
			t.Error(err)
			return
		}

		current, _, err := opensearchclient.GetIndexVersionsOfAlias(client, ctx, kind)
		if err != nil {
			t.Error(err)
			return
		}
		if current != expectedCurrent {
			t.Error(current, expectedCurrent)
			return
		}
	}
}

func testMappingAddElements(config configuration.Config) func(t *testing.T) {
	return func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		p, err := k.NewProducer(ctx, config.KafkaUrl, "device-types", true)
		if err != nil {
			t.Error(err)
			return
		}
		t.Run("create dt1", createTestDeviceType(p, "dt1", "n1", "dc1"))
		t.Run("create dt2", createTestDeviceType(p, "dt2", "n2", "dc2"))
	}
}

func createTestDeviceType(p *k.Producer, id string, name string, deviceClassId string) func(t *testing.T) {
	return func(t *testing.T) {
		msg, cmd := getDtTestObj(id, map[string]interface{}{
			"id":              id,
			"name":            name,
			"device_class_id": deviceClassId,
		})
		err := p.Produce(cmd.Id, msg)
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func testMappingReadBy(config configuration.Config, field string, value string, expectedResultCount int) func(t *testing.T) {
	return func(t *testing.T) {
		requestBody := &bytes.Buffer{}
		err := json.NewEncoder(requestBody).Encode(model.QueryMessage{
			Resource: "device-types",
			Find: &model.QueryFind{
				Filter: &model.Selection{
					Condition: model.ConditionConfig{
						Feature:   "features." + field,
						Operation: model.QueryEqualOperation,
						Value:     value,
					},
				},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}

		req, err := http.NewRequest("POST", "http://localhost:"+config.ServerPort+"/v2/query", requestBody)
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
			t.Error(resp.StatusCode, string(temp))
			return
		}

		var actual []interface{}
		err = json.NewDecoder(resp.Body).Decode(&actual)
		if err != nil {
			t.Error(err)
			return
		}

		if len(actual) != expectedResultCount {
			t.Error(len(actual), actual, expectedResultCount)
			return
		}
	}
}

func testMappingSearch(config configuration.Config, search string, expectedResultCount int) func(t *testing.T) {
	return func(t *testing.T) {
		requestBody := &bytes.Buffer{}
		err := json.NewEncoder(requestBody).Encode(model.QueryMessage{
			Resource: "device-types",
			Find: &model.QueryFind{
				Search: search,
			},
		})
		if err != nil {
			t.Error(err)
			return
		}

		req, err := http.NewRequest("POST", "http://localhost:"+config.ServerPort+"/v2/query", requestBody)
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
			t.Error(resp.StatusCode, string(temp))
			return
		}

		var actual []interface{}
		err = json.NewDecoder(resp.Body).Decode(&actual)
		if err != nil {
			t.Error(err)
			return
		}

		if len(actual) != expectedResultCount {
			t.Error(len(actual), actual, expectedResultCount)
			return
		}
	}
}

func startTestServer(config configuration.Config, mapping string, ctx context.Context) func(t *testing.T) {
	return func(t *testing.T) {
		mappingConfig := configuration.ConfigStruct{}
		err := json.Unmarshal([]byte(mapping), &mappingConfig)
		if err != nil {
			t.Error(err)
			return
		}

		config.IndexTypeMapping = mappingConfig.IndexTypeMapping
		config.Resources = mappingConfig.Resources
		config.ResourceList = []string{}
		for resource := range config.Resources {
			config.ResourceList = append(config.ResourceList, resource)
		}

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
	}

}

func testUpdateIndex(config configuration.Config, mapping string) func(t *testing.T) {
	return func(t *testing.T) {
		mappingConfig := configuration.ConfigStruct{}
		err := json.Unmarshal([]byte(mapping), &mappingConfig)
		if err != nil {
			t.Error(err)
			return
		}

		config.IndexTypeMapping = mappingConfig.IndexTypeMapping
		config.Resources = mappingConfig.Resources
		config.ResourceList = []string{}
		for resource := range config.Resources {
			config.ResourceList = append(config.ResourceList, resource)
		}

		err = opensearchclient.UpdateIndexes(config, "device-types")
		if err != nil {
			t.Error(err)
			return
		}
	}

}

const confV1 = `{
	"resources": {
		"device-types":{
            "features":[
                {"Name": "name", "Path": "$.device_type.name+"},
                {"Name": "device_class_id", "Path": "$.device_type.device_class_id+"}
            ],
            "initial_group_rights":{"admin": "rwxa", "user": "rx"}
        }
	},
    "index_type_mapping": {
        "device-types": {
			"features": {
            	"name":         {"type": "keyword", "copy_to": "feature_search"}
        	}
		}
    }
}`

const confV2 = `{
	"resources": {
		"device-types":{
            "features":[
                {"Name": "name", "Path": "$.device_type.name+"},
                {"Name": "device_class_id", "Path": "$.device_type.device_class_id+"}
            ],
            "initial_group_rights":{"admin": "rwxa", "user": "rx"}
        }
	},
    "index_type_mapping": {
        "device-types": {
			"features": {
            	"name":         {"type": "keyword", "copy_to": "feature_search"},
				"device_class_id":  {"type": "keyword", "copy_to": "feature_search"}
			}
        }
    }
}`
