//go:build !ci
// +build !ci

/*
 * Copyright 2023 InfAI (CC SES)
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
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/query"
	"github.com/SENERGY-Platform/permission-search/lib/worker"
	"github.com/olivere/elastic/v7"
	"reflect"
	"sync"
	"time"
)

func Example() {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config, q, w, err := getTestEnv(ctx, wg, nil)
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
	//not found
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

func ExampleSearch() {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config, q, w, err := getTestEnv(ctx, wg, nil)
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
	//not found
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
	config, q, w, err := getTestEnv(ctx, wg, nil)
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
	config, q, w, err := getTestEnv(ctx, wg, nil)
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
	config, q, w, err := getTestEnv(ctx, wg, nil)
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
	fmt.Println(q.CheckUserOrGroupFromAuthToken(createTestToken("nope", []string{}), "device-types", "check3", "a"))

	fmt.Println(q.CheckUserOrGroupFromAuthToken(createTestToken("nope", []string{"user"}), "device-types", "check3", "a"))

	fmt.Println(q.CheckUserOrGroupFromAuthToken(createTestToken("nope", []string{"user"}), "device-types", "check3", "r"))

	fmt.Println(q.CheckUserOrGroupFromAuthToken(createTestToken("testOwner", []string{"user"}), "device-types", "check3", "a"))

	fmt.Println(q.CheckUserOrGroupFromAuthToken(createTestToken("testOwner", []string{"user"}), "device-types", "check3", "ra"))
	fmt.Println(q.CheckUserOrGroupFromAuthToken(createTestToken("nope", []string{"user"}), "device-types", "check3", "ra"))

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
	config, q, w, err := getTestEnv(ctx, wg, nil)
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
	config, q, w, err := getTestEnv(ctx, wg, nil)
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
	config, q, w, err := getTestEnv(ctx, wg, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	initDb(config, w)

	time.Sleep(1 * time.Second)
	result, err := q.GetList(createTestToken("testOwner", []string{}), "device-types", model.QueryListCommons{
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
	result, err = q.GetList(createTestToken("testOwner", []string{"user"}), "device-types", model.QueryListCommons{
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
	result, err = q.GetList(createTestToken("testOwner", []string{}), "device-types", model.QueryListCommons{
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
	result, err = q.GetList(createTestToken("testOwner", []string{}), "device-types", model.QueryListCommons{
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
	config, q, w, err := getTestEnv(ctx, wg, nil)
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
