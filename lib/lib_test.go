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
	"testing"

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
	_, err := GetClient().DeleteByQuery("devicetype").Query(elastic.NewMatchAllQuery()).Do(context.Background())
	if err != nil {
		panic(err)
	}
	_, err = client.Flush().Index("devicetype").Do(context.Background())
	if err != nil {
		panic(err)
	}
	err = UpdateFeatures("devicetype", test, testCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = UpdateFeatures("devicetype", foo1, foo1Cmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = UpdateFeatures("devicetype", foo2, foo2Cmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = UpdateFeatures("devicetype", bar, barCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = UpdateFeatures("devicetype", zway, zwayCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = client.Flush().Index("devicetype").Do(context.Background())
	if err != nil {
		panic(err)
	}
	e, err := GetResourceEntry("devicetype", "test")
	fmt.Println(err, e.Resource)
	e, err = GetResourceEntry("devicetype", "foo1")
	fmt.Println(err, e.Resource)
	e, err = GetResourceEntry("devicetype", "foo2")
	fmt.Println(err, e.Resource)
	e, err = GetResourceEntry("devicetype", "zway")
	fmt.Println(err, e.Resource)
	e, err = GetResourceEntry("devicetype", "bar")
	fmt.Println(err, e.Resource)

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
	Example()
	_, err := client.Flush().Index("devicetype").Do(context.Background())
	if err != nil {
		panic(err)
	}
	query := elastic.NewBoolQuery().Should(
		elastic.NewTermQuery("admin_users", "testOwner"),
		elastic.NewTermQuery("read_users", "testOwner"),
		elastic.NewTermQuery("write_users", "testOwner"),
		elastic.NewTermQuery("execute_users", "testOwner"))
	result, err := GetClient().Search().Index("devicetype").Query(query).Do(context.Background())
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
	//foo2
	//foo1
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
	_, err := GetClient().DeleteByQuery("devicetype").Query(elastic.NewMatchAllQuery()).Do(context.Background())
	if err != nil {
		panic(err)
	}
	_, err = client.Flush().Index("devicetype").Do(context.Background())
	if err != nil {
		panic(err)
	}
	err = UpdateFeatures("devicetype", msg, cmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = DeleteUser("testOwner")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = DeleteGroupRight("devicetype", "del", "user")
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = client.Flush().Index("devicetype").Do(context.Background())
	if err != nil {
		panic(err)
	}
	query := elastic.NewMatchAllQuery()
	result, err := GetClient().Search().Index("devicetype").Query(query).Do(context.Background())
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
	//del [testOwner] [testOwner] [testOwner] [testOwner] [admin] [admin] [admin] [admin]
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
	err := UpdateFeatures("devicetype", msg, cmd)
	msg, cmd = getDtTestObj("nodel", map[string]interface{}{
		"name":        "ZWay-SwitchMultilevel",
		"description": "desc",
		"maintenance": []string{},
		"services":    []map[string]interface{}{},
		"vendor":      map[string]interface{}{"name": "vendor"},
	})
	_, err = GetClient().DeleteByQuery("devicetype").Query(elastic.NewMatchAllQuery()).Do(context.Background())
	if err != nil {
		panic(err)
	}
	_, err = client.Flush().Index("devicetype").Do(context.Background())
	if err != nil {
		panic(err)
	}
	err = UpdateFeatures("devicetype", msg, cmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = DeleteFeatures("devicetype", CommandWrapper{Id: "del1"})
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = client.Flush().Index("devicetype").Do(context.Background())
	if err != nil {
		panic(err)
	}
	query := elastic.NewMatchAllQuery()
	result, err := GetClient().Search().Index("devicetype").Query(query).Do(context.Background())
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
	rights, err := GetRightsToAdministrate("devicetype", "nope", []string{})
	fmt.Println(len(rights), err)

	rights, err = GetRightsToAdministrate("devicetype", "testOwner", []string{})
	fmt.Println(len(rights), err)

	rights, err = GetRightsToAdministrate("devicetype", "testOwner", []string{"nope"})
	fmt.Println(len(rights), err)

	rights, err = GetRightsToAdministrate("devicetype", "testOwner", []string{"nope", "admin"})
	fmt.Println(len(rights), err)

	rights, err = GetRightsToAdministrate("devicetype", "testOwner", []string{"admin"})
	fmt.Println(len(rights), err)

	rights, err = GetRightsToAdministrate("devicetype", "nope", []string{"nope", "admin"})
	fmt.Println(len(rights), err)

	rights, err = GetRightsToAdministrate("devicetype", "nope", []string{"admin"})
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
	_, err := GetClient().DeleteByQuery("devicetype").Query(elastic.NewMatchAllQuery()).Do(context.Background())
	if err != nil {
		panic(err)
	}
	_, err = client.Flush().Index("devicetype").Do(context.Background())
	if err != nil {
		panic(err)
	}
	UpdateFeatures("devicetype", test, testCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = client.Flush().Index("devicetype").Do(context.Background())

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(CheckUserOrGroup("devicetype", "check3", "nope", []string{}, "a"))

	fmt.Println(CheckUserOrGroup("devicetype", "check3", "nope", []string{"user"}, "a"))

	fmt.Println(CheckUserOrGroup("devicetype", "check3", "nope", []string{"user"}, "r"))

	fmt.Println(CheckUserOrGroup("devicetype", "check3", "testOwner", []string{"user"}, "a"))

	fmt.Println(CheckUserOrGroup("devicetype", "check3", "testOwner", []string{"user"}, "ra"))
	fmt.Println(CheckUserOrGroup("devicetype", "check3", "nope", []string{"user"}, "ra"))

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

	result, err := GetFullListForUserOrGroup("devicetype", "testOwner", []string{}, "r")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println(r["name"])
	}
	//Output:
	//<nil>
	//foo1
	//test
	//ZWay-SwitchMultilevel
	//foo2
}

func ExampleGetListForUserOrGroup() {
	initDb()

	result, err := GetListForUserOrGroup("devicetype", "testOwner", []string{}, "r", "20", "0")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println(r["name"])
	}
	result, err = GetListForUserOrGroup("devicetype", "testOwner", []string{}, "r", "3", "0")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println(r["name"])
	}
	result, err = GetListForUserOrGroup("devicetype", "testOwner", []string{}, "r", "3", "3")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println(r["name"])
	}
	//Output:
	//<nil>
	//foo1
	//test
	//ZWay-SwitchMultilevel
	//foo2
	//<nil>
	//foo1
	//test
	//ZWay-SwitchMultilevel
	//<nil>
	//foo2
}

func ExampleGetOrderedListForUserOrGroup() {
	initDb()

	result, err := GetOrderedListForUserOrGroup("devicetype", "testOwner", []string{}, "r", "20", "0", "name", true)
	fmt.Println(err)
	for _, r := range result {
		fmt.Println(r["name"])
	}
	result, err = GetOrderedListForUserOrGroup("devicetype", "testOwner", []string{}, "r", "20", "0", "name", false)
	fmt.Println(err)
	for _, r := range result {
		fmt.Println(r["name"])
	}
	result, err = GetOrderedListForUserOrGroup("devicetype", "testOwner", []string{}, "r", "3", "0", "name", true)
	fmt.Println(err)
	for _, r := range result {
		fmt.Println(r["name"])
	}
	result, err = GetOrderedListForUserOrGroup("devicetype", "testOwner", []string{}, "r", "3", "3", "name", true)
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

	result, err := SearchRightsToAdministrate("devicetype", "testOwner", []string{}, "z", "20", "0")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println("found: ", r.ResourceId)
	}
	result, err = SearchRightsToAdministrate("devicetype", "testOwner", []string{}, "zway", "20", "0")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println("found: ", r.ResourceId)
	}
	result, err = SearchRightsToAdministrate("devicetype", "testOwner", []string{}, "zway switch", "20", "0")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println("found: ", r.ResourceId)
	}
	result, err = SearchRightsToAdministrate("devicetype", "testOwner", []string{}, "switch", "20", "0")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println("found: ", r.ResourceId)
	}

	result, err = SearchRightsToAdministrate("devicetype", "testOwner", []string{}, "nope", "20", "0")
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

func ExampleSelectByFieldAll() {
	initDb()

	result, err := SelectByFieldAll("devicetype", "service", "foo2Service", "testOwner", []string{}, "r")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println("found: ", r["name"])
	}
	result, err = SelectByFieldAll("devicetype", "service", "foo", "testOwner", []string{}, "r")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println("found: ", r["name"])
	}
	result, err = SelectByFieldAll("devicetype", "maintenance", "something", "testOwner", []string{}, "r")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println("found: ", r["name"])
	}
	result, err = SelectByFieldAll("devicetype", "maintenance", "so", "testOwner", []string{}, "r")
	fmt.Println(err)
	for _, r := range result {
		fmt.Println("found: ", r["name"])
	}

	//Output:
	//<nil>
	//found:  foo2
	//<nil>
	//<nil>
	//found:  test
	//<nil>
}

func ExampleJsonpath() {
	jsonStr := `{  
   "command":"PUT",
   "processmodel":{  
      "process":{  
         "definitions":{  
            "process":{  
               "startEvent":{  
                  "outgoing":{  
                     "__prefix":"bpmn",
                     "__text":"SequenceFlow_0mfiuuu"
                  },
                  "_id":"StartEvent_1",
                  "__prefix":"bpmn"
               },
               "endEvent":{  
                  "incoming":{  
                     "__prefix":"bpmn",
                     "__text":"SequenceFlow_0mfiuuu"
                  },
                  "_id":"EndEvent_0oe34u0",
                  "__prefix":"bpmn"
               },
               "sequenceFlow":{  
                  "_id":"SequenceFlow_0mfiuuu",
                  "_sourceRef":"StartEvent_1",
                  "_targetRef":"EndEvent_0oe34u0",
                  "__prefix":"bpmn"
               },
               "_id":"FooBar1",
               "_isExecutable":"true",
               "__prefix":"bpmn"
            },
            "BPMNDiagram":{  
               "BPMNPlane":{  
                  "BPMNShape":[  
                     {  
                        "Bounds":{  
                           "_x":"173",
                           "_y":"102",
                           "_width":"36",
                           "_height":"36",
                           "__prefix":"dc"
                        },
                        "_id":"_BPMNShape_StartEvent_2",
                        "_bpmnElement":"StartEvent_1",
                        "__prefix":"bpmndi"
                     },
                     {  
                        "Bounds":{  
                           "_x":"231",
                           "_y":"102",
                           "_width":"36",
                           "_height":"36",
                           "__prefix":"dc"
                        },
                        "BPMNLabel":{  
                           "Bounds":{  
                              "_x":"249",
                              "_y":"138",
                              "_width":"0",
                              "_height":"0",
                              "__prefix":"dc"
                           },
                           "__prefix":"bpmndi"
                        },
                        "_id":"EndEvent_0oe34u0_di",
                        "_bpmnElement":"EndEvent_0oe34u0",
                        "__prefix":"bpmndi"
                     }
                  ],
                  "BPMNEdge":{  
                     "waypoint":[  
                        {  
                           "_xsi:type":"dc:Point",
                           "_x":"209",
                           "_y":"120",
                           "__prefix":"di"
                        },
                        {  
                           "_xsi:type":"dc:Point",
                           "_x":"231",
                           "_y":"120",
                           "__prefix":"di"
                        }
                     ],
                     "BPMNLabel":{  
                        "Bounds":{  
                           "_x":"220",
                           "_y":"95",
                           "_width":"0",
                           "_height":"0",
                           "__prefix":"dc"
                        },
                        "__prefix":"bpmndi"
                     },
                     "_id":"SequenceFlow_0mfiuuu_di",
                     "_bpmnElement":"SequenceFlow_0mfiuuu",
                     "__prefix":"bpmndi"
                  },
                  "_id":"BPMNPlane_1",
                  "_bpmnElement":"FooBar1",
                  "__prefix":"bpmndi"
               },
               "_id":"BPMNDiagram_1",
               "__prefix":"bpmndi"
            },
            "_xmlns:xsi":"http://www.w3.org/2001/XMLSchema-instance",
            "_xmlns:bpmn":"http://www.omg.org/spec/BPMN/20100524/MODEL",
            "_xmlns:bpmndi":"http://www.omg.org/spec/BPMN/20100524/DI",
            "_xmlns:dc":"http://www.omg.org/spec/DD/20100524/DC",
            "_xmlns:di":"http://www.omg.org/spec/DD/20100524/DI",
            "_id":"Definitions_1",
            "_targetNamespace":"http://bpmn.io/schema/bpmn",
            "__prefix":"bpmn"
         }
      },
      "svg":{  
         "svg":{  
            "defs":{  
               "marker":[  
                  {  
                     "path":{  
                        "_d":"M 1 5 L 11 10 L 1 15 Z",
                        "_style":"stroke-width: 1; stroke-linecap: round; stroke-dasharray: 10000, 1;",
                        "_fill":"#000000"
                     },
                     "_viewBox":"0 0 20 20",
                     "_markerWidth":"10",
                     "_markerHeight":"10",
                     "_orient":"auto",
                     "_refX":"11",
                     "_refY":"10",
                     "_id":"markerSjhrbfypa4"
                  },
                  {  
                     "circle":{  
                        "_cx":"6",
                        "_cy":"6",
                        "_r":"3.5",
                        "_style":"stroke-width: 1; stroke-linecap: round; stroke-dasharray: 10000, 1;",
                        "_fill":"#ffffff",
                        "_stroke":"#000000"
                     },
                     "_viewBox":"0 0 20 20",
                     "_markerWidth":"20",
                     "_markerHeight":"20",
                     "_orient":"auto",
                     "_refX":"6",
                     "_refY":"6",
                     "_id":"markerSjhrbfypa6"
                  },
                  {  
                     "path":{  
                        "_d":"m 1 5 l 0 -3 l 7 3 l -7 3 z",
                        "_fill":"#ffffff",
                        "_stroke":"#000000",
                        "_style":"stroke-width: 1; stroke-linecap: butt; stroke-dasharray: 10000, 1;"
                     },
                     "_viewBox":"0 0 20 20",
                     "_markerWidth":"20",
                     "_markerHeight":"20",
                     "_orient":"auto",
                     "_refX":"8.5",
                     "_refY":"5",
                     "_id":"markerSjhrbfypa8"
                  },
                  {  
                     "path":{  
                        "_d":"M 11 5 L 1 10 L 11 15",
                        "_fill":"none",
                        "_stroke":"#000000",
                        "_style":"stroke-width: 1.5; stroke-linecap: round; stroke-dasharray: 10000, 1;"
                     },
                     "_viewBox":"0 0 20 20",
                     "_markerWidth":"10",
                     "_markerHeight":"10",
                     "_orient":"auto",
                     "_refX":"1",
                     "_refY":"10",
                     "_id":"markerSjhrbfypaa"
                  },
                  {  
                     "path":{  
                        "_d":"M 1 5 L 11 10 L 1 15",
                        "_fill":"none",
                        "_stroke":"#000000",
                        "_style":"stroke-width: 1.5; stroke-linecap: round; stroke-dasharray: 10000, 1;"
                     },
                     "_viewBox":"0 0 20 20",
                     "_markerWidth":"10",
                     "_markerHeight":"10",
                     "_orient":"auto",
                     "_refX":"12",
                     "_refY":"10",
                     "_id":"markerSjhrbfypac"
                  },
                  {  
                     "path":{  
                        "_d":"M 0 10 L 8 6 L 16 10 L 8 14 Z",
                        "_fill":"#ffffff",
                        "_stroke":"#000000",
                        "_style":"stroke-width: 1; stroke-linecap: round; stroke-dasharray: 10000, 1;"
                     },
                     "_viewBox":"0 0 20 20",
                     "_markerWidth":"10",
                     "_markerHeight":"10",
                     "_orient":"auto",
                     "_refX":"-1",
                     "_refY":"10",
                     "_id":"markerSjhrbfypae"
                  },
                  {  
                     "path":{  
                        "_d":"M 1 4 L 5 16",
                        "_fill":"#000000",
                        "_stroke":"#000000",
                        "_style":"stroke-width: 1; stroke-linecap: round; stroke-dasharray: 10000, 1;"
                     },
                     "_viewBox":"0 0 20 20",
                     "_markerWidth":"10",
                     "_markerHeight":"10",
                     "_orient":"auto",
                     "_refX":"-5",
                     "_refY":"10",
                     "_id":"markerSjhrbfypag"
                  }
               ]
            },
            "g":[  
               {  
                  "g":{  
                     "g":{  
                        "circle":{  
                           "_cx":"18",
                           "_cy":"18",
                           "_r":"18",
                           "_stroke":"#000000",
                           "_fill":"#ffffff",
                           "_style":"stroke-width: 2;"
                        },
                        "_class":"djs-visual"
                     },
                     "rect":[  
                        {  
                           "_x":"0",
                           "_y":"0",
                           "_width":"36",
                           "_height":"36",
                           "_fill":"none",
                           "_stroke":"#ffffff",
                           "_style":"stroke-opacity: 0; stroke-width: 15;",
                           "_class":"djs-hit"
                        },
                        {  
                           "_x":"-6",
                           "_y":"-6",
                           "_width":"48",
                           "_height":"48",
                           "_fill":"none",
                           "_style":"",
                           "_class":"djs-outline"
                        }
                     ],
                     "_data-element-id":"StartEvent_1",
                     "_transform":"matrix(1,0,0,1,173,102)",
                     "_style":"display: block;",
                     "_class":"djs-element djs-shape"
                  },
                  "_xmlns":"http://www.w3.org/2000/svg",
                  "_class":"djs-group"
               },
               {  
                  "g":{  
                     "g":{  
                        "text":{  
                           "tspan":{  
                              "_x":"45",
                              "_y":"0"
                           },
                           "_class":" djs-label",
                           "_style":"font-family: Arial, sans-serif; font-size: 11px;",
                           "_transform":"matrix(1,0,0,1,0,0)"
                        },
                        "_class":"djs-visual"
                     },
                     "rect":[  
                        {  
                           "_x":"0",
                           "_y":"0",
                           "_width":"0",
                           "_height":"0",
                           "_fill":"none",
                           "_stroke":"#ffffff",
                           "_style":"stroke-opacity: 0; stroke-width: 15;",
                           "_class":"djs-hit"
                        },
                        {  
                           "_x":"-6",
                           "_y":"-6",
                           "_width":"12",
                           "_height":"12",
                           "_fill":"none",
                           "_style":"",
                           "_class":"djs-outline"
                        }
                     ],
                     "_data-element-id":"StartEvent_1_label",
                     "_class":"djs-element djs-shape",
                     "_transform":"matrix(1,0,0,1,191,138)",
                     "_style":"display: none;"
                  },
                  "_xmlns":"http://www.w3.org/2000/svg",
                  "_class":"djs-group"
               },
               {  
                  "g":{  
                     "g":{  
                        "circle":{  
                           "_cx":"18",
                           "_cy":"18",
                           "_r":"18",
                           "_stroke":"#000000",
                           "_fill":"#ffffff",
                           "_style":"stroke-width: 4;"
                        },
                        "_class":"djs-visual"
                     },
                     "rect":[  
                        {  
                           "_x":"0",
                           "_y":"0",
                           "_width":"36",
                           "_height":"36",
                           "_fill":"none",
                           "_stroke":"#ffffff",
                           "_style":"stroke-opacity: 0; stroke-width: 15;",
                           "_class":"djs-hit"
                        },
                        {  
                           "_x":"-6",
                           "_y":"-6",
                           "_width":"48",
                           "_height":"48",
                           "_fill":"none",
                           "_style":"",
                           "_class":"djs-outline"
                        }
                     ],
                     "_data-element-id":"EndEvent_0oe34u0",
                     "_transform":"matrix(1,0,0,1,231,102)",
                     "_style":"display: block;",
                     "_class":"djs-element djs-shape"
                  },
                  "_xmlns":"http://www.w3.org/2000/svg",
                  "_class":"djs-group"
               },
               {  
                  "g":{  
                     "g":{  
                        "text":{  
                           "tspan":{  
                              "_x":"45",
                              "_y":"0"
                           },
                           "_class":" djs-label",
                           "_style":"font-family: Arial, sans-serif; font-size: 11px;",
                           "_transform":"matrix(1,0,0,1,0,0)"
                        },
                        "_class":"djs-visual"
                     },
                     "rect":[  
                        {  
                           "_x":"0",
                           "_y":"0",
                           "_width":"0",
                           "_height":"0",
                           "_fill":"none",
                           "_stroke":"#ffffff",
                           "_style":"stroke-opacity: 0; stroke-width: 15;",
                           "_class":"djs-hit"
                        },
                        {  
                           "_x":"-6",
                           "_y":"-6",
                           "_width":"12",
                           "_height":"12",
                           "_fill":"none",
                           "_style":"",
                           "_class":"djs-outline"
                        }
                     ],
                     "_data-element-id":"EndEvent_0oe34u0_label",
                     "_class":"djs-element djs-shape",
                     "_transform":"matrix(1,0,0,1,249,138)",
                     "_style":"display: none;"
                  },
                  "_xmlns":"http://www.w3.org/2000/svg",
                  "_class":"djs-group"
               },
               {  
                  "g":{  
                     "g":{  
                        "path":{  
                           "_d":"m  209,120L231,120 ",
                           "_fill":"none",
                           "_stroke":"#000000",
                           "_style":"stroke-width: 2; stroke-linejoin: round; marker-end: url(\"#markerSjhrbfypa4\");"
                        },
                        "_class":"djs-visual"
                     },
                     "polyline":{  
                        "_points":"209,120 231,120 ",
                        "_fill":"none",
                        "_stroke":"#ffffff",
                        "_style":"stroke-opacity: 0; stroke-width: 15;",
                        "_class":"djs-hit"
                     },
                     "rect":{  
                        "_x":"203",
                        "_y":"114",
                        "_width":"34",
                        "_height":"12",
                        "_fill":"none",
                        "_style":"",
                        "_class":"djs-outline"
                     },
                     "_data-element-id":"SequenceFlow_0mfiuuu",
                     "_class":"djs-element djs-connection",
                     "_style":"display: block;"
                  },
                  "_xmlns":"http://www.w3.org/2000/svg",
                  "_class":"djs-group"
               },
               {  
                  "g":{  
                     "g":{  
                        "text":{  
                           "tspan":{  
                              "_x":"45",
                              "_y":"0"
                           },
                           "_class":" djs-label",
                           "_style":"font-family: Arial, sans-serif; font-size: 11px;",
                           "_transform":"matrix(1,0,0,1,0,0)"
                        },
                        "_class":"djs-visual"
                     },
                     "rect":[  
                        {  
                           "_x":"0",
                           "_y":"0",
                           "_width":"0",
                           "_height":"0",
                           "_fill":"none",
                           "_stroke":"#ffffff",
                           "_style":"stroke-opacity: 0; stroke-width: 15;",
                           "_class":"djs-hit"
                        },
                        {  
                           "_x":"-6",
                           "_y":"-6",
                           "_width":"12",
                           "_height":"12",
                           "_fill":"none",
                           "_style":"",
                           "_class":"djs-outline"
                        }
                     ],
                     "_data-element-id":"SequenceFlow_0mfiuuu_label",
                     "_class":"djs-element djs-shape",
                     "_transform":"matrix(1,0,0,1,220,95)",
                     "_style":"display: none;"
                  },
                  "_xmlns":"http://www.w3.org/2000/svg",
                  "_class":"djs-group"
               }
            ],
            "_xmlns":"http://www.w3.org/2000/svg",
            "_xmlns:xlink":"http://www.w3.org/1999/xlink",
            "_width":"106",
            "_height":"48",
            "_viewBox":"167 96 106 48",
            "_version":"1.1"
         }
      },
      "date":1527577962056,
      "_id":"5b0812f2d10ea4001614e002"
   },
   "id":"5b0812f2d10ea4001614e002"
}`

	fmt.Println(UseJsonPath([]byte(jsonStr), "$.processmodel.svg+"))

	//Output:
	//
}

func initDb() {
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

	_, err := GetClient().DeleteByQuery("devicetype").Query(elastic.NewMatchAllQuery()).Do(context.Background())
	if err != nil {
		panic(err)
	}
	_, err = client.Flush().Index("devicetype").Do(context.Background())
	if err != nil {
		panic(err)
	}
	err = UpdateFeatures("devicetype", test, testCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = UpdateFeatures("devicetype", foo1, foo1Cmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = UpdateFeatures("devicetype", foo2, foo2Cmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = UpdateFeatures("devicetype", bar, barCmd)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = UpdateFeatures("devicetype", zway, zwayCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = client.Flush().Index("devicetype").Do(context.Background())
	if err != nil {
		panic(err)
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
		code = m.Run()
	}()
	os.Exit(code)
}
