/*
 * Copyright 2018 InfAI (CC SES)
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
	"fmt"
	"github.com/SENERGY-Platform/permission-search/lib/api"
	"github.com/SENERGY-Platform/permission-search/lib/auth"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/query"
	"github.com/SENERGY-Platform/permission-search/lib/tests/docker"
	"github.com/SENERGY-Platform/permission-search/lib/worker"
	k "github.com/SENERGY-Platform/permission-search/lib/worker/kafka"
	"github.com/opensearch-project/opensearch-go/opensearchutil"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"sync"
	"testing"
)

var OpenSearch = docker.OpenSearch
var Zookeeper = docker.Zookeeper
var Kafka = docker.Kafka

func getDtTestObj(id string, dt map[string]interface{}) (msg []byte, command model.CommandWrapper) {
	text := `{
		"command": "PUT",
		"id": "%s",
		"owner": "testOwner",
		"device_type": %s
	}`
	dtStr, err := json.Marshal(dt)
	if err != nil {
		fmt.Println(err)
		return
	}
	msg = []byte(fmt.Sprintf(text, id, string(dtStr)))
	err = json.Unmarshal(msg, &command)
	if err != nil {
		fmt.Println(err)
		return
	}
	return
}

func initDb(config configuration.Config, worker *worker.Worker) {
	config.MaxRetry = 3
	test, testCmd := getDtTestObj("test", map[string]interface{}{
		"name":        "test",
		"description": "something",
		"maintenance": []string{"something", "onotherthing"},
		"services":    []map[string]interface{}{{"id": "serviceTest1"}, {"id": "serviceTest2"}},
		"vendor":      map[string]interface{}{"name": "vendor"},
	})
	foo1, foo1Cmd := getDtTestObj("foo1", map[string]interface{}{
		"name":        "foo1",
		"description": "foo1Desc",
		"maintenance": []string{},
		"services":    []map[string]interface{}{{"id": "foo1Service"}},
		"vendor":      map[string]interface{}{"name": "foo1Vendor"},
	})
	foo2, foo2Cmd := getDtTestObj("foo2", map[string]interface{}{
		"name":        "foo2",
		"description": "foo2Desc",
		"maintenance": []string{},
		"services":    []map[string]interface{}{{"id": "foo2Service"}},
		"vendor":      map[string]interface{}{"name": "foo2Vendor"},
	})
	bar, barCmd := getDtTestObj("test", map[string]interface{}{
		"name":        "test",
		"description": "changedDesc",
		"maintenance": []string{"something", "different"},
		"services":    []map[string]interface{}{{"id": "serviceTest1"}, {"id": "serviceTest3"}},
		"vendor":      map[string]interface{}{"name": "chengedvendor"},
	})
	//ZWay-SwitchMultilevel
	zway, zwayCmd := getDtTestObj("zway", map[string]interface{}{
		"name":        "ZWay-SwitchMultilevel",
		"description": "desc",
		"maintenance": []string{},
		"services":    []map[string]interface{}{},
		"vendor":      map[string]interface{}{"name": "vendor"},
	})

	_, err := worker.GetClient().DeleteByQuery([]string{"device-types"}, opensearchutil.NewJSONReader(map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
	}))
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = worker.GetClient().Indices.Flush(worker.GetClient().Indices.Flush.WithIndex("device-types"))
	if err != nil {
		fmt.Println(err)
		return
	}
	err = worker.UpdateFeatures("device-types", test, testCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = worker.UpdateFeatures("device-types", foo1, foo1Cmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = worker.UpdateFeatures("device-types", foo2, foo2Cmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = worker.UpdateFeatures("device-types", bar, barCmd)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = worker.UpdateFeatures("device-types", zway, zwayCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = worker.GetClient().Indices.Flush(worker.GetClient().Indices.Flush.WithIndex("device-types"))
	if err != nil {
		fmt.Println(err)
		return
	}
}

func getTestEnv(ctx context.Context, wg *sync.WaitGroup, t *testing.T) (config configuration.Config, q *query.Query, w *worker.Worker, err error) {
	config, err = configuration.LoadConfig("./../config.json")
	if err != nil {
		return config, q, w, err
	}
	config.LogDeprecatedCallsToFile = ""
	if t != nil {
		config.FatalErrHandler = func(v ...interface{}) {
			log.Println("TEST-ERROR:", v)
			t.Log(v...)
		}
	} else {
		config.FatalErrHandler = func(v ...interface{}) {
			log.Println(v...)
			return
		}
	}

	config.OpenSearchInsecureSkipVerify = true
	config.OpenSearchUsername = "admin"
	config.OpenSearchPassword = "01J1iEnT#>kE"

	_, ip, err := OpenSearch(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.OpenSearchUrls = "https://" + ip + ":9200"

	q, err = query.New(config)
	if err != nil {
		return config, q, w, err
	}
	w, err = worker.New(ctx, config, q)
	if err != nil {
		return config, q, w, err
	}
	return
}

func getTestEnvWithApi(ctx context.Context, wg *sync.WaitGroup, t *testing.T) (config configuration.Config, q *query.Query, w *worker.Worker, err error) {
	config, q, w, err = getTestEnv(ctx, wg, t)
	if err != nil {
		return config, q, w, err
	}
	handler, err := api.GetRouter(config, q, nil)
	if err != nil {
		return config, q, w, err
	}
	server := httptest.NewServer(handler)
	serverUrl, err := url.Parse(server.URL)
	if err != nil {
		return config, q, w, err
	}
	wg.Add(1)
	go func() {
		<-ctx.Done()
		server.Close()
		wg.Done()
	}()
	config.ServerPort = serverUrl.Port()
	return
}

const testTokenUser = "testOwner"
const testtoken = `Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiIwOGM0N2E4OC0yYzc5LTQyMGYtODEwNC02NWJkOWViYmU0MWUiLCJleHAiOjE1NDY1MDcyMzMsIm5iZiI6MCwiaWF0IjoxNTQ2NTA3MTczLCJpc3MiOiJodHRwOi8vbG9jYWxob3N0OjgwMDEvYXV0aC9yZWFsbXMvbWFzdGVyIiwiYXVkIjoiZnJvbnRlbmQiLCJzdWIiOiJ0ZXN0T3duZXIiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJmcm9udGVuZCIsIm5vbmNlIjoiOTJjNDNjOTUtNzViMC00NmNmLTgwYWUtNDVkZDk3M2I0YjdmIiwiYXV0aF90aW1lIjoxNTQ2NTA3MDA5LCJzZXNzaW9uX3N0YXRlIjoiNWRmOTI4ZjQtMDhmMC00ZWI5LTliNjAtM2EwYWUyMmVmYzczIiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJ1c2VyIl19LCJyZXNvdXJjZV9hY2Nlc3MiOnsibWFzdGVyLXJlYWxtIjp7InJvbGVzIjpbInZpZXctcmVhbG0iLCJ2aWV3LWlkZW50aXR5LXByb3ZpZGVycyIsIm1hbmFnZS1pZGVudGl0eS1wcm92aWRlcnMiLCJpbXBlcnNvbmF0aW9uIiwiY3JlYXRlLWNsaWVudCIsIm1hbmFnZS11c2VycyIsInF1ZXJ5LXJlYWxtcyIsInZpZXctYXV0aG9yaXphdGlvbiIsInF1ZXJ5LWNsaWVudHMiLCJxdWVyeS11c2VycyIsIm1hbmFnZS1ldmVudHMiLCJtYW5hZ2UtcmVhbG0iLCJ2aWV3LWV2ZW50cyIsInZpZXctdXNlcnMiLCJ2aWV3LWNsaWVudHMiLCJtYW5hZ2UtYXV0aG9yaXphdGlvbiIsIm1hbmFnZS1jbGllbnRzIiwicXVlcnktZ3JvdXBzIl19LCJhY2NvdW50Ijp7InJvbGVzIjpbIm1hbmFnZS1hY2NvdW50IiwibWFuYWdlLWFjY291bnQtbGlua3MiLCJ2aWV3LXByb2ZpbGUiXX19LCJyb2xlcyI6WyJ1c2VyIl19.ykpuOmlpzj75ecSI6cHbCATIeY4qpyut2hMc1a67Ycg`

const adminTokenUser = "admin"
const admintoken = `Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiIwOGM0N2E4OC0yYzc5LTQyMGYtODEwNC02NWJkOWViYmU0MWUiLCJleHAiOjE1NDY1MDcyMzMsIm5iZiI6MCwiaWF0IjoxNTQ2NTA3MTczLCJpc3MiOiJodHRwOi8vbG9jYWxob3N0OjgwMDEvYXV0aC9yZWFsbXMvbWFzdGVyIiwiYXVkIjoiZnJvbnRlbmQiLCJzdWIiOiJhZG1pbiIsInR5cCI6IkJlYXJlciIsImF6cCI6ImZyb250ZW5kIiwibm9uY2UiOiI5MmM0M2M5NS03NWIwLTQ2Y2YtODBhZS00NWRkOTczYjRiN2YiLCJhdXRoX3RpbWUiOjE1NDY1MDcwMDksInNlc3Npb25fc3RhdGUiOiI1ZGY5MjhmNC0wOGYwLTRlYjktOWI2MC0zYTBhZTIyZWZjNzMiLCJhY3IiOiIwIiwiYWxsb3dlZC1vcmlnaW5zIjpbIioiXSwicmVhbG1fYWNjZXNzIjp7InJvbGVzIjpbInVzZXIiLCJhZG1pbiJdfSwicmVzb3VyY2VfYWNjZXNzIjp7Im1hc3Rlci1yZWFsbSI6eyJyb2xlcyI6WyJ2aWV3LXJlYWxtIiwidmlldy1pZGVudGl0eS1wcm92aWRlcnMiLCJtYW5hZ2UtaWRlbnRpdHktcHJvdmlkZXJzIiwiaW1wZXJzb25hdGlvbiIsImNyZWF0ZS1jbGllbnQiLCJtYW5hZ2UtdXNlcnMiLCJxdWVyeS1yZWFsbXMiLCJ2aWV3LWF1dGhvcml6YXRpb24iLCJxdWVyeS1jbGllbnRzIiwicXVlcnktdXNlcnMiLCJtYW5hZ2UtZXZlbnRzIiwibWFuYWdlLXJlYWxtIiwidmlldy1ldmVudHMiLCJ2aWV3LXVzZXJzIiwidmlldy1jbGllbnRzIiwibWFuYWdlLWF1dGhvcml6YXRpb24iLCJtYW5hZ2UtY2xpZW50cyIsInF1ZXJ5LWdyb3VwcyJdfSwiYWNjb3VudCI6eyJyb2xlcyI6WyJtYW5hZ2UtYWNjb3VudCIsIm1hbmFnZS1hY2NvdW50LWxpbmtzIiwidmlldy1wcm9maWxlIl19fSwicm9sZXMiOlsidXNlciIsImFkbWluIl19.ggcFFFEsjwdfSzEFzmZt_m6W4IiSQub2FRhZVfWttDI`

const secendOwnerTokenUser = "secondOwner"
const secondOwnerToken = `Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiIwOGM0N2E4OC0yYzc5LTQyMGYtODEwNC02NWJkOWViYmU0MWUiLCJleHAiOjE1NDY1MDcyMzMsIm5iZiI6MCwiaWF0IjoxNTQ2NTA3MTczLCJpc3MiOiJodHRwOi8vbG9jYWxob3N0OjgwMDEvYXV0aC9yZWFsbXMvbWFzdGVyIiwiYXVkIjoiZnJvbnRlbmQiLCJzdWIiOiJzZWNvbmRPd25lciIsInR5cCI6IkJlYXJlciIsImF6cCI6ImZyb250ZW5kIiwibm9uY2UiOiI5MmM0M2M5NS03NWIwLTQ2Y2YtODBhZS00NWRkOTczYjRiN2YiLCJhdXRoX3RpbWUiOjE1NDY1MDcwMDksInNlc3Npb25fc3RhdGUiOiI1ZGY5MjhmNC0wOGYwLTRlYjktOWI2MC0zYTBhZTIyZWZjNzMiLCJhY3IiOiIwIiwiYWxsb3dlZC1vcmlnaW5zIjpbIioiXSwicmVhbG1fYWNjZXNzIjp7InJvbGVzIjpbInVzZXIiXX0sInJlc291cmNlX2FjY2VzcyI6eyJtYXN0ZXItcmVhbG0iOnsicm9sZXMiOlsidmlldy1yZWFsbSIsInZpZXctaWRlbnRpdHktcHJvdmlkZXJzIiwibWFuYWdlLWlkZW50aXR5LXByb3ZpZGVycyIsImltcGVyc29uYXRpb24iLCJjcmVhdGUtY2xpZW50IiwibWFuYWdlLXVzZXJzIiwicXVlcnktcmVhbG1zIiwidmlldy1hdXRob3JpemF0aW9uIiwicXVlcnktY2xpZW50cyIsInF1ZXJ5LXVzZXJzIiwibWFuYWdlLWV2ZW50cyIsIm1hbmFnZS1yZWFsbSIsInZpZXctZXZlbnRzIiwidmlldy11c2VycyIsInZpZXctY2xpZW50cyIsIm1hbmFnZS1hdXRob3JpemF0aW9uIiwibWFuYWdlLWNsaWVudHMiLCJxdWVyeS1ncm91cHMiXX0sImFjY291bnQiOnsicm9sZXMiOlsibWFuYWdlLWFjY291bnQiLCJtYW5hZ2UtYWNjb3VudC1saW5rcyIsInZpZXctcHJvZmlsZSJdfX0sInJvbGVzIjpbInVzZXIiXX0.cq8YeUuR0jSsXCEzp634fTzNbGkq_B8KbVrwBPgceJ4`

func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
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
	return testRequestWithToken(config, testtoken, method, path, body, expectedStatusCode, expected)
}

func testRequestWithToken(config configuration.Config, token string, method string, path string, body interface{}, expectedStatusCode int, expected interface{}) func(t *testing.T) {
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
		req.Header.Set("Authorization", token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
			return
		}
		if resp.StatusCode != expectedStatusCode {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}

		if expected != nil {
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
				t.Error("\n", string(a), "\n", string(e))
				return
			}
		}
	}
}

func getTestAspectResult(id string) map[string]interface{} {
	return getTestAspectResultWithPermissionHolders(id, []string{"testOwner"}, false)
}

func getTestAspectResultWithPermissionHolders(id string, userList []string, shared bool) map[string]interface{} {
	//map[creator:testOwner id:aaspect name:aaspect_name permissions:map[a:true r:true w:true x:true] shared:false
	sort.Strings(userList)
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
		"raw": map[string]interface{}{
			"name":     id + "_name",
			"rdf_type": "aspect_type",
		},
		"permission_holders": map[string][]string{
			"admin_users":   userList,
			"execute_users": userList,
			"read_users":    userList,
			"write_users":   userList,
		},
		"shared": shared,
	}
}

func saveTestDevice(w *worker.Worker, resource string, id string, fields map[string]interface{}) func(t *testing.T) {
	return func(t *testing.T) {
		msg, cmd, err := getDeviceTestObj(id, fields)
		if err != nil {
			t.Error(err)
			return
		}
		err = w.UpdateFeatures(resource, msg, cmd)
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func getDeviceTestObj(id string, obj interface{}) (msg []byte, command model.CommandWrapper, err error) {
	text := `{
		"command": "PUT",
		"id": "%s",
		"owner": "testOwner",
		"device": %s
	}`
	dtStr, err := json.Marshal(obj)
	if err != nil {
		return msg, command, err
	}
	msg = []byte(fmt.Sprintf(text, id, string(dtStr)))
	err = json.Unmarshal(msg, &command)
	if err != nil {
		return msg, command, err
	}
	return msg, command, err
}

func createTestToken(user string, groups []string) auth.Token {
	return auth.Token{
		Token:       "",
		Sub:         user,
		RealmAccess: map[string][]string{"roles": groups},
	}
}

func getAspectTestObj(id string, obj map[string]interface{}) (msg []byte, command model.CommandWrapper, err error) {
	text := `{
		"command": "PUT",
		"id": "%s",
		"owner": "testOwner",
		"aspect": %s
	}`
	dtStr, err := json.Marshal(obj)
	if err != nil {
		return msg, command, err
	}
	msg = []byte(fmt.Sprintf(text, id, string(dtStr)))
	err = json.Unmarshal(msg, &command)
	if err != nil {
		return msg, command, err
	}
	return msg, command, err
}
