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
	"encoding/json"
	"fmt"
	"github.com/ory/dockertest"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"testing"
	"time"

	"context"

	"reflect"

	elastic "github.com/olivere/elastic/v7"
)

func Example() {
	Config.ElasticRetry = 3
	test, testCmd := getDtTestObj("test", map[string]interface{}{
		"name":        "test",
		"description": "desc",
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
	_, err := GetClient().DeleteByQuery("device-types").Query(elastic.NewMatchAllQuery()).Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = client.Flush().Index("device-types").Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	err = UpdateFeatures("device-types", test, testCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = UpdateFeatures("device-types", foo1, foo1Cmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = UpdateFeatures("device-types", foo2, foo2Cmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = UpdateFeatures("device-types", bar, barCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = UpdateFeatures("device-types", zway, zwayCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = client.Flush().Index("device-types").Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	e, err := GetResourceEntry("device-types", "test")
	fmt.Println(err, e.Resource)
	e, err = GetResourceEntry("device-types", "foo1")
	fmt.Println(err, e.Resource)
	e, err = GetResourceEntry("device-types", "foo2")
	fmt.Println(err, e.Resource)
	e, err = GetResourceEntry("device-types", "zway")
	fmt.Println(err, e.Resource)
	_, err = GetResourceEntry("device-types", "bar")
	fmt.Println(err)

	//Output:
	//<nil> test
	//<nil> foo1
	//<nil> foo2
	//<nil> zway
	//elastic: Error 404 (Not Found)
}

func getDtTestObj(id string, dt map[string]interface{}) (msg []byte, command CommandWrapper) {
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

func ExampleSearch() {
	time.Sleep(1 * time.Second)
	Example()
	_, err := client.Flush().Index("device-types").Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	query := elastic.NewBoolQuery().Should(
		elastic.NewTermQuery("admin_users", "testOwner"),
		elastic.NewTermQuery("read_users", "testOwner"),
		elastic.NewTermQuery("write_users", "testOwner"),
		elastic.NewTermQuery("execute_users", "testOwner"))
	time.Sleep(1 * time.Second)
	result, err := GetClient().Search().Index("device-types").Query(query).Sort("resource", true).Do(context.Background())
	fmt.Println(err)

	var entity Entry
	if result != nil {
		for _, item := range result.Each(reflect.TypeOf(entity)) {
			if t, ok := item.(Entry); ok {
				fmt.Println(t.Resource)
			}
		}
	}

	//Output:
	//<nil> test
	//<nil> foo1
	//<nil> foo2
	//<nil> zway
	//elastic: Error 404 (Not Found)
	//<nil>
	//foo1
	//foo2
	//test
	//zway

}

func ExampleDeleteUser() {
	Config.ElasticRetry = 3
	msg, cmd := getDtTestObj("del", map[string]interface{}{
		"name":        "ZWay-SwitchMultilevel",
		"description": "desc",
		"maintenance": []string{},
		"services":    []map[string]interface{}{},
		"vendor":      map[string]interface{}{"name": "vendor"},
	})
	_, err := GetClient().DeleteByQuery("device-types").Query(elastic.NewMatchAllQuery()).Do(context.Background())
	if err != nil {
		fmt.Println(err)
		debug.PrintStack()
		return
	}
	_, err = client.Flush().Index("device-types").Do(context.Background())
	if err != nil {
		fmt.Println(err)
		debug.PrintStack()
		return
	}
	err = UpdateFeatures("device-types", msg, cmd)
	if err != nil {
		fmt.Println(err)
		debug.PrintStack()
		return
	}
	time.Sleep(1 * time.Second)
	err = DeleteUser("testOwner")
	if err != nil {
		fmt.Println(err)
		debug.PrintStack()
		return
	}
	time.Sleep(1 * time.Second)
	err = DeleteGroupRight("device-types", "del", "user")
	if err != nil {
		fmt.Println(err)
		debug.PrintStack()
		return
	}
	time.Sleep(1 * time.Second)
	_, err = client.Flush().Index("device-types").Do(context.Background())
	if err != nil {
		fmt.Println(err)
		debug.PrintStack()
		return
	}
	query := elastic.NewMatchAllQuery()
	result, err := GetClient().Search().Index("device-types").Query(query).Do(context.Background())
	fmt.Println(err)
	var entity Entry
	if result != nil {
		for _, item := range result.Each(reflect.TypeOf(entity)) {
			if t, ok := item.(Entry); ok {
				fmt.Println(t.Resource, t.ReadUsers, t.WriteUsers, t.ExecuteUsers, t.AdminUsers, t.ReadGroups, t.WriteGroups, t.ExecuteGroups, t.AdminGroups)
			}
		}
	}

	//Output:
	//<nil>
	//del [] [] [] [] [admin] [admin] [admin] [admin]
}

func ExampleDeleteFeatures() {
	Config.ElasticRetry = 3
	msg, cmd := getDtTestObj("del1", map[string]interface{}{
		"name":        "ZWay-SwitchMultilevel",
		"description": "desc",
		"maintenance": []string{},
		"services":    []map[string]interface{}{},
		"vendor":      map[string]interface{}{"name": "vendor"},
	})
	err := UpdateFeatures("device-types", msg, cmd)
	msg, cmd = getDtTestObj("nodel", map[string]interface{}{
		"name":        "ZWay-SwitchMultilevel",
		"description": "desc",
		"maintenance": []string{},
		"services":    []map[string]interface{}{},
		"vendor":      map[string]interface{}{"name": "vendor"},
	})
	_, err = GetClient().DeleteByQuery("device-types").Query(elastic.NewMatchAllQuery()).Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = client.Flush().Index("device-types").Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	time.Sleep(1 * time.Second)
	err = UpdateFeatures("device-types", msg, cmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	time.Sleep(1 * time.Second)
	err = DeleteFeatures("device-types", CommandWrapper{Id: "del1"})
	if err != nil {
		fmt.Println(err)
		return
	}
	time.Sleep(1 * time.Second)
	_, err = client.Flush().Index("device-types").Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	query := elastic.NewMatchAllQuery()
	result, err := GetClient().Search().Index("device-types").Query(query).Do(context.Background())
	fmt.Println(err)
	var entity Entry
	if result != nil {
		for _, item := range result.Each(reflect.TypeOf(entity)) {
			if t, ok := item.(Entry); ok {
				fmt.Println(t.Resource)
			}
		}
	}

	//Output:
	//<nil>
	//nodel
}

func ExampleGetRightsToAdministrate() {
	initDb()
	time.Sleep(1 * time.Second)
	rights, err := GetRightsToAdministrate("device-types", "nope", []string{})
	fmt.Println(len(rights), err)

	rights, err = GetRightsToAdministrate("device-types", "testOwner", []string{})
	fmt.Println(len(rights), err)

	rights, err = GetRightsToAdministrate("device-types", "testOwner", []string{"nope"})
	fmt.Println(len(rights), err)

	rights, err = GetRightsToAdministrate("device-types", "testOwner", []string{"nope", "admin"})
	fmt.Println(len(rights), err)

	rights, err = GetRightsToAdministrate("device-types", "testOwner", []string{"admin"})
	fmt.Println(len(rights), err)

	rights, err = GetRightsToAdministrate("device-types", "nope", []string{"nope", "admin"})
	fmt.Println(len(rights), err)

	rights, err = GetRightsToAdministrate("device-types", "nope", []string{"admin"})
	fmt.Println(len(rights), err)

	//Output:
	//0 <nil>
	//4 <nil>
	//4 <nil>
	//4 <nil>
	//4 <nil>
	//4 <nil>
	//4 <nil>
}

func ExampleCheckUserOrGroup() {
	Config.ElasticRetry = 3
	test, testCmd := getDtTestObj("check3", map[string]interface{}{
		"name":        "test",
		"description": "desc",
		"maintenance": []string{"something", "onotherthing"},
		"services":    []map[string]interface{}{{"id": "serviceTest1"}, {"id": "serviceTest2"}},
		"vendor":      map[string]interface{}{"name": "vendor"},
	})
	_, err := GetClient().DeleteByQuery("device-types").Query(elastic.NewMatchAllQuery()).Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = client.Flush().Index("device-types").Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	time.Sleep(1 * time.Second)
	err = UpdateFeatures("device-types", test, testCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = client.Flush().Index("device-types").Do(context.Background())

	if err != nil {
		fmt.Println(err)
		return
	}
	time.Sleep(1 * time.Second)
	fmt.Println(CheckUserOrGroup("device-types", "check3", "nope", []string{}, "a"))

	fmt.Println(CheckUserOrGroup("device-types", "check3", "nope", []string{"user"}, "a"))

	fmt.Println(CheckUserOrGroup("device-types", "check3", "nope", []string{"user"}, "r"))

	fmt.Println(CheckUserOrGroup("device-types", "check3", "testOwner", []string{"user"}, "a"))

	fmt.Println(CheckUserOrGroup("device-types", "check3", "testOwner", []string{"user"}, "ra"))
	fmt.Println(CheckUserOrGroup("device-types", "check3", "nope", []string{"user"}, "ra"))

	//Output:
	//access denied
	//access denied
	//<nil>
	//<nil>
	//<nil>
	//access denied
}

func ExampleGetFullListForUserOrGroup() {
	initDb()
	time.Sleep(1 * time.Second)
	result, err := GetFullListForUserOrGroup("device-types", "testOwner", []string{}, "r")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println(r["name"])
	}
	//Output:
	//<nil>
	//foo1
	//foo2
	//test
	//ZWay-SwitchMultilevel
}

func ExampleGetListForUserOrGroup() {
	initDb()
	time.Sleep(1 * time.Second)
	result, err := GetListForUserOrGroup("device-types", "testOwner", []string{}, "r", "20", "0")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println(r["name"])
	}
	result, err = GetListForUserOrGroup("device-types", "testOwner", []string{}, "r", "3", "0")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println(r["name"])
	}
	result, err = GetListForUserOrGroup("device-types", "testOwner", []string{}, "r", "3", "3")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println(r["name"])
	}
	//Output:
	//<nil>
	//foo1
	//foo2
	//test
	//ZWay-SwitchMultilevel
	//<nil>
	//foo1
	//foo2
	//test
	//<nil>
	//ZWay-SwitchMultilevel
}

func ExampleGetOrderedListForUserOrGroup() {
	initDb()
	time.Sleep(1 * time.Second)
	result, err := GetOrderedListForUserOrGroup("device-types", "testOwner", []string{}, "r", "20", "0", "name", true)
	fmt.Println(err)
	for _, r := range result {
		fmt.Println(r["name"])
	}
	result, err = GetOrderedListForUserOrGroup("device-types", "testOwner", []string{}, "r", "20", "0", "name", false)
	fmt.Println(err)
	for _, r := range result {
		fmt.Println(r["name"])
	}
	result, err = GetOrderedListForUserOrGroup("device-types", "testOwner", []string{}, "r", "3", "0", "name", true)
	fmt.Println(err)
	for _, r := range result {
		fmt.Println(r["name"])
	}
	result, err = GetOrderedListForUserOrGroup("device-types", "testOwner", []string{}, "r", "3", "3", "name", true)
	fmt.Println(err)
	for _, r := range result {
		fmt.Println(r["name"])
	}
	//Output:
	//<nil>
	//ZWay-SwitchMultilevel
	//foo1
	//foo2
	//test
	//<nil>
	//test
	//foo2
	//foo1
	//ZWay-SwitchMultilevel
	//<nil>
	//ZWay-SwitchMultilevel
	//foo1
	//foo2
	//<nil>
	//test
}

func ExampleSearchRightsToAdministrate() {
	initDb()
	time.Sleep(1 * time.Second)
	result, err := SearchRightsToAdministrate("device-types", "testOwner", []string{}, "z", "20", "0")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println("found: ", r.ResourceId)
	}
	result, err = SearchRightsToAdministrate("device-types", "testOwner", []string{}, "zway", "20", "0")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println("found: ", r.ResourceId)
	}
	result, err = SearchRightsToAdministrate("device-types", "testOwner", []string{}, "zway switch", "20", "0")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println("found: ", r.ResourceId)
	}
	result, err = SearchRightsToAdministrate("device-types", "testOwner", []string{}, "switch", "20", "0")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println("found: ", r.ResourceId)
	}

	result, err = SearchRightsToAdministrate("device-types", "testOwner", []string{}, "nope", "20", "0")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println("found: ", r.ResourceId)
	}

	//Output:
	//<nil>
	//found:  zway
	//<nil>
	//found:  zway
	//<nil>
	//found:  zway
	//<nil>
	//found:  zway
	//<nil>
}

func initDb() {
	Config.ElasticRetry = 3
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

	_, err := GetClient().DeleteByQuery("device-types").Query(elastic.NewMatchAllQuery()).Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = client.Flush().Index("device-types").Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	err = UpdateFeatures("device-types", test, testCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = UpdateFeatures("device-types", foo1, foo1Cmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = UpdateFeatures("device-types", foo2, foo2Cmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = UpdateFeatures("device-types", bar, barCmd)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = UpdateFeatures("device-types", zway, zwayCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = client.Flush().Index("device-types").Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
}

func Elasticsearch(pool *dockertest.Pool) (closer func(), hostPort string, ipAddress string, err error) {
	log.Println("start elasticsearch")
	repo, err := pool.Run("docker.elastic.co/elasticsearch/elasticsearch", "7.6.1", []string{"discovery.type=single-node"})
	if err != nil {
		return func() {}, "", "", err
	}
	hostPort = repo.GetPort("9200/tcp")
	err = pool.Retry(func() error {
		log.Println("try elastic connection...")
		_, err := http.Get("http://" + repo.Container.NetworkSettings.IPAddress + ":9200/_cluster/health")
		return err
	})
	if err != nil {
		log.Println(err)
	}
	return func() { repo.Close() }, hostPort, repo.Container.NetworkSettings.IPAddress, err
}

func TestMain(m *testing.M) {
	var code int
	func() {
		pool, err := dockertest.NewPool("")
		if err != nil {
			fmt.Println(err)
			code = 1
			return
		}
		close, port, _, err := Elasticsearch(pool)
		if err != nil {
			fmt.Println(err)
			code = 1
			return
		}
		defer close()
		err = LoadConfig("./../config.json")
		if err != nil {
			fmt.Println(err)
			code = 1
			return
		}
		Config.ElasticUrl = "http://localhost:" + port

		ctx := context.Background()
		client, err := elastic.NewClient(elastic.SetURL(Config.ElasticUrl), elastic.SetRetrier(newRetrier()))
		if err != nil {
			fmt.Println(err)
			return
		}
		for kind := range Config.Resources {
			err = createIndex(kind, client, ctx)
			if err != nil {
				fmt.Println(err)
				return
			}
		}
		client.Stop()
		code = m.Run()
	}()
	os.Exit(code)
}
