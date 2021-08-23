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
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/query"
	"github.com/SENERGY-Platform/permission-search/lib/worker"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"log"
	"net/http"
	"sync"
	"time"

	"context"

	"reflect"

	elastic "github.com/olivere/elastic/v7"
)

func Example() {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config, q, w, err := getTestEnv(ctx, wg)
	if err != nil {
		fmt.Println(err)
		return
	}
	config.ElasticRetry = 3
	example(w, q)

	//Output:
	//<nil> test
	//<nil> foo1
	//<nil> foo2
	//<nil> zway
	//elastic: Error 404 (Not Found)
}

func example(w *worker.Worker, q *query.Query) {

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
	_, err := q.GetClient().DeleteByQuery("device-types").Query(elastic.NewMatchAllQuery()).Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = q.GetClient().Flush().Index("device-types").Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	err = w.UpdateFeatures("device-types", test, testCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = w.UpdateFeatures("device-types", foo1, foo1Cmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = w.UpdateFeatures("device-types", foo2, foo2Cmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = w.UpdateFeatures("device-types", bar, barCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = w.UpdateFeatures("device-types", zway, zwayCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = q.GetClient().Flush().Index("device-types").Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	e, _, err := q.GetResourceEntry("device-types", "test")
	fmt.Println(err, e.Resource)
	e, _, err = q.GetResourceEntry("device-types", "foo1")
	fmt.Println(err, e.Resource)
	e, _, err = q.GetResourceEntry("device-types", "foo2")
	fmt.Println(err, e.Resource)
	e, _, err = q.GetResourceEntry("device-types", "zway")
	fmt.Println(err, e.Resource)
	_, _, err = q.GetResourceEntry("device-types", "bar")
	fmt.Println(err)
}

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

func ExampleSearch() {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config, q, w, err := getTestEnv(ctx, wg)
	if err != nil {
		fmt.Println(err)
		return
	}
	config.ElasticRetry = 3

	time.Sleep(1 * time.Second)
	example(w, q)
	_, err = q.GetClient().Flush().Index("device-types").Do(context.Background())
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
	result, err := q.GetClient().Search().Index("device-types").Query(query).Sort("resource", true).Do(context.Background())
	fmt.Println(err)

	var entity model.Entry
	if result != nil {
		for _, item := range result.Each(reflect.TypeOf(entity)) {
			if t, ok := item.(model.Entry); ok {
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

func ExampleDeleteFeatures() {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config, q, w, err := getTestEnv(ctx, wg)
	if err != nil {
		fmt.Println(err)
		return
	}

	config.ElasticRetry = 3
	msg, cmd := getDtTestObj("del1", map[string]interface{}{
		"name":        "ZWay-SwitchMultilevel",
		"description": "desc",
		"maintenance": []string{},
		"services":    []map[string]interface{}{},
		"vendor":      map[string]interface{}{"name": "vendor"},
	})
	err = w.UpdateFeatures("device-types", msg, cmd)
	msg, cmd = getDtTestObj("nodel", map[string]interface{}{
		"name":        "ZWay-SwitchMultilevel",
		"description": "desc",
		"maintenance": []string{},
		"services":    []map[string]interface{}{},
		"vendor":      map[string]interface{}{"name": "vendor"},
	})
	_, err = q.GetClient().DeleteByQuery("device-types").Query(elastic.NewMatchAllQuery()).Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = q.GetClient().Flush().Index("device-types").Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	time.Sleep(1 * time.Second)
	err = w.UpdateFeatures("device-types", msg, cmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	time.Sleep(1 * time.Second)
	err = w.DeleteFeatures("device-types", model.CommandWrapper{Id: "del1"})
	if err != nil {
		fmt.Println(err)
		return
	}
	time.Sleep(1 * time.Second)
	_, err = q.GetClient().Flush().Index("device-types").Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	query := elastic.NewMatchAllQuery()
	result, err := q.GetClient().Search().Index("device-types").Query(query).Do(context.Background())
	fmt.Println(err)
	var entity model.Entry
	if result != nil {
		for _, item := range result.Each(reflect.TypeOf(entity)) {
			if t, ok := item.(model.Entry); ok {
				fmt.Println(t.Resource)
			}
		}
	}

	//Output:
	//<nil>
	//nodel
}

func ExampleGetRightsToAdministrate() {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config, q, w, err := getTestEnv(ctx, wg)
	if err != nil {
		fmt.Println(err)
		return
	}
	initDb(config, w)

	time.Sleep(1 * time.Second)
	rights, err := q.GetRightsToAdministrate("device-types", "nope", []string{})
	fmt.Println(len(rights), err)

	rights, err = q.GetRightsToAdministrate("device-types", "testOwner", []string{})
	fmt.Println(len(rights), err)

	rights, err = q.GetRightsToAdministrate("device-types", "testOwner", []string{"nope"})
	fmt.Println(len(rights), err)

	rights, err = q.GetRightsToAdministrate("device-types", "testOwner", []string{"nope", "admin"})
	fmt.Println(len(rights), err)

	rights, err = q.GetRightsToAdministrate("device-types", "testOwner", []string{"admin"})
	fmt.Println(len(rights), err)

	rights, err = q.GetRightsToAdministrate("device-types", "nope", []string{"nope", "admin"})
	fmt.Println(len(rights), err)

	rights, err = q.GetRightsToAdministrate("device-types", "nope", []string{"admin"})
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
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config, q, w, err := getTestEnv(ctx, wg)
	if err != nil {
		fmt.Println(err)
		return
	}

	config.ElasticRetry = 3
	test, testCmd := getDtTestObj("check3", map[string]interface{}{
		"name":        "test",
		"description": "desc",
		"maintenance": []string{"something", "onotherthing"},
		"services":    []map[string]interface{}{{"id": "serviceTest1"}, {"id": "serviceTest2"}},
		"vendor":      map[string]interface{}{"name": "vendor"},
	})
	_, err = q.GetClient().DeleteByQuery("device-types").Query(elastic.NewMatchAllQuery()).Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = q.GetClient().Flush().Index("device-types").Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	time.Sleep(1 * time.Second)
	err = w.UpdateFeatures("device-types", test, testCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = q.GetClient().Flush().Index("device-types").Do(context.Background())

	if err != nil {
		fmt.Println(err)
		return
	}
	time.Sleep(1 * time.Second)
	fmt.Println(q.CheckUserOrGroup("device-types", "check3", "nope", []string{}, "a"))

	fmt.Println(q.CheckUserOrGroup("device-types", "check3", "nope", []string{"user"}, "a"))

	fmt.Println(q.CheckUserOrGroup("device-types", "check3", "nope", []string{"user"}, "r"))

	fmt.Println(q.CheckUserOrGroup("device-types", "check3", "testOwner", []string{"user"}, "a"))

	fmt.Println(q.CheckUserOrGroup("device-types", "check3", "testOwner", []string{"user"}, "ra"))
	fmt.Println(q.CheckUserOrGroup("device-types", "check3", "nope", []string{"user"}, "ra"))

	//Output:
	//access denied
	//access denied
	//<nil>
	//<nil>
	//<nil>
	//access denied
}

func ExampleGetFullListForUserOrGroup() {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config, q, w, err := getTestEnv(ctx, wg)
	if err != nil {
		fmt.Println(err)
		return
	}
	initDb(config, w)

	time.Sleep(1 * time.Second)
	result, err := q.GetFullListForUserOrGroup("device-types", "testOwner", []string{}, "r")
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
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config, q, w, err := getTestEnv(ctx, wg)
	if err != nil {
		fmt.Println(err)
		return
	}
	initDb(config, w)

	time.Sleep(1 * time.Second)
	result, err := q.GetListForUserOrGroup("device-types", "testOwner", []string{}, "r", "20", "0")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println(r["name"])
	}
	result, err = q.GetListForUserOrGroup("device-types", "testOwner", []string{}, "r", "3", "0")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println(r["name"])
	}
	result, err = q.GetListForUserOrGroup("device-types", "testOwner", []string{}, "r", "3", "3")
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
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config, q, w, err := getTestEnv(ctx, wg)
	if err != nil {
		fmt.Println(err)
		return
	}
	initDb(config, w)

	time.Sleep(1 * time.Second)
	result, err := q.GetOrderedListForUserOrGroup("device-types", "testOwner", []string{}, model.QueryListCommons{
		Limit:    20,
		Offset:   0,
		Rights:   "r",
		SortBy:   "name",
		SortDesc: false,
	})
	fmt.Println(err)
	for _, r := range result {
		fmt.Println(r["name"])
	}
	result, err = q.GetOrderedListForUserOrGroup("device-types", "testOwner", []string{}, model.QueryListCommons{
		Limit:    20,
		Offset:   0,
		Rights:   "r",
		SortBy:   "name",
		SortDesc: true,
	})
	fmt.Println(err)
	for _, r := range result {
		fmt.Println(r["name"])
	}
	result, err = q.GetOrderedListForUserOrGroup("device-types", "testOwner", []string{}, model.QueryListCommons{
		Limit:    3,
		Offset:   0,
		Rights:   "r",
		SortBy:   "name",
		SortDesc: false,
	})
	fmt.Println(err)
	for _, r := range result {
		fmt.Println(r["name"])
	}
	result, err = q.GetOrderedListForUserOrGroup("device-types", "testOwner", []string{}, model.QueryListCommons{
		Limit:    3,
		Offset:   3,
		Rights:   "r",
		SortBy:   "name",
		SortDesc: false,
	})
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
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config, q, w, err := getTestEnv(ctx, wg)
	if err != nil {
		fmt.Println(err)
		return
	}
	initDb(config, w)

	time.Sleep(1 * time.Second)
	result, err := q.SearchRightsToAdministrate("device-types", "testOwner", []string{}, "z", "20", "0")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println("found: ", r.ResourceId)
	}
	result, err = q.SearchRightsToAdministrate("device-types", "testOwner", []string{}, "zway", "20", "0")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println("found: ", r.ResourceId)
	}
	result, err = q.SearchRightsToAdministrate("device-types", "testOwner", []string{}, "zway switch", "20", "0")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println("found: ", r.ResourceId)
	}
	result, err = q.SearchRightsToAdministrate("device-types", "testOwner", []string{}, "switch", "20", "0")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println("found: ", r.ResourceId)
	}

	result, err = q.SearchRightsToAdministrate("device-types", "testOwner", []string{}, "nope", "20", "0")
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

func initDb(config configuration.Config, worker *worker.Worker) {
	config.ElasticRetry = 3
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

	_, err := worker.GetClient().DeleteByQuery("device-types").Query(elastic.NewMatchAllQuery()).Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = worker.GetClient().Flush().Index("device-types").Do(context.Background())
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
	_, err = worker.GetClient().Flush().Index("device-types").Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
}

func elasticsearch(ctx context.Context, wg *sync.WaitGroup) (hostPort string, ipAddress string, err error) {
	log.Println("start elasticsearch")
	pool, err := dockertest.NewPool("")
	if err != nil {
		return "", "", err
	}

	container, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "docker.elastic.co/elasticsearch/elasticsearch",
		Tag:        "7.6.1",
		Env: []string{
			"discovery.type=single-node",
			"path.data=/opt/elasticsearch/volatile/data",
			"path.logs=/opt/elasticsearch/volatile/logs",
		},
	}, func(config *docker.HostConfig) {
		config.Tmpfs = map[string]string{
			"/opt/elasticsearch/volatile/data": "rw",
			"/opt/elasticsearch/volatile/logs": "rw",
			"/tmp":                             "rw",
		}
	})

	if err != nil {
		return "", "", err
	}
	wg.Add(1)
	go func() {
		<-ctx.Done()
		log.Println("DEBUG: remove container " + container.Container.Name)
		container.Close()
		wg.Done()
	}()
	hostPort = container.GetPort("9200/tcp")
	err = pool.Retry(func() error {
		log.Println("try elastic connection...")
		_, err := http.Get("http://" + container.Container.NetworkSettings.IPAddress + ":9200/_cluster/health")
		return err
	})
	if err != nil {
		log.Println(err)
	}
	return hostPort, container.Container.NetworkSettings.IPAddress, err
}

func getTestEnv(ctx context.Context, wg *sync.WaitGroup) (config configuration.Config, q *query.Query, w *worker.Worker, err error) {
	config, err = configuration.LoadConfig("./../config.json")
	if err != nil {
		return config, q, w, err
	}
	port, _, err := elasticsearch(ctx, wg)
	if err != nil {
		return config, q, w, err
	}
	config.ElasticUrl = "http://localhost:" + port
	q, err = query.New(config)
	if err != nil {
		return config, q, w, err
	}
	w = worker.New(config, q)
	return
}
