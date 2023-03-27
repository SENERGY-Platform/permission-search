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
	"fmt"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/olivere/elastic/v7"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestDeviceGroup(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, q, w, err := getTestEnv(ctx, wg, t)
	if err != nil {
		fmt.Println(err)
		return
	}

	resource := "device-groups"
	_, err = q.GetClient().DeleteByQuery(resource).Query(elastic.NewMatchAllQuery()).Do(context.Background())
	if err != nil {
		t.Error(err)
		return
	}
	_, err = q.GetClient().Flush().Index(resource).Do(context.Background())
	if err != nil {
		t.Error(err)
		return
	}

	msg, cmd, err := getDeviceGroupTestObj("g1", map[string]interface{}{
		"name":       "g1_name",
		"image":      "g1_image",
		"device_ids": []string{"d1", "d2"},
		"foo":        "bar",
		"bar":        42,
		"criteria": []interface{}{
			map[string]interface{}{
				"function_id":     "f1",
				"aspect_id":       "a1",
				"device_class_id": "",
				"interaction":     "request",
			},
			map[string]interface{}{
				"function_id": "f1",
				"aspect_id":   "a2",
				"interaction": "event",
			},
			map[string]interface{}{
				"function_id":     "f2",
				"aspect_id":       "",
				"device_class_id": "dc1",
				"interaction":     "request",
			},
			map[string]interface{}{
				"function_id":     "f3",
				"device_class_id": "dc1",
			},
		},
		"criteria_short": []interface{}{
			"f1_a1__request",
			"f1_a2__event",
			"f2__dc1_request",
			"f3__dc1_",
		},
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, msg, cmd)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	e, _, err := q.GetResourceEntry(resource, "g1")
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(e.Features["name"], "g1_name") {
		t.Error(e)
		return
	}

	expected := []map[string]interface{}{{
		"id":             "g1",
		"name":           "g1_name",
		"image":          "g1_image",
		"attributes":     nil,
		"attribute_list": nil,
		"device_ids":     []interface{}{"d1", "d2"},
		"criteria": []interface{}{
			map[string]interface{}{
				"function_id":     "f1",
				"aspect_id":       "a1",
				"device_class_id": "",
				"interaction":     "request",
			},
			map[string]interface{}{
				"function_id": "f1",
				"aspect_id":   "a2",
				"interaction": "event",
			},
			map[string]interface{}{
				"function_id":     "f2",
				"aspect_id":       "",
				"device_class_id": "dc1",
				"interaction":     "request",
			},
			map[string]interface{}{
				"function_id":     "f3",
				"device_class_id": "dc1",
			},
		},
		"criteria_short": []interface{}{
			"f1_a1__request",
			"f1_a2__event",
			"f2__dc1_request",
			"f3__dc1_",
		},
		"creator": "testOwner",
		"permissions": map[string]bool{
			"a": true,
			"r": true,
			"w": true,
			"x": true,
		},
		"permission_holders": map[string][]string{
			"admin_users":   {"testOwner"},
			"execute_users": {"testOwner"},
			"read_users":    {"testOwner"},
			"write_users":   {"testOwner"},
		},
		"shared": false,
	}}

	result, err := q.GetList(createTestToken("testOwner", []string{"user"}), resource, model.QueryListCommons{
		Limit:    3,
		Offset:   0,
		Rights:   "r",
		SortBy:   "name",
		SortDesc: false,
	})
	if err != nil {
		t.Error(err)
		return
	}
	if len(result) != 1 {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result, expected) {
		j, _ := json.Marshal(result)
		t.Error(string(j))
		return
	}

	filter := model.Selection{
		And: []model.Selection{
			{
				Condition: model.ConditionConfig{
					Feature:   "features.criteria_short",
					Operation: model.QueryEqualOperation,
					Value:     "f1_a1__request",
				},
			},
			{
				Condition: model.ConditionConfig{
					Feature:   "features.criteria_short",
					Operation: model.QueryEqualOperation,
					Value:     "f1_a2__event",
				},
			},
		},
	}
	if err != nil {
		t.Error(err)
		return
	}
	result, err = q.GetListWithSelection(
		createTestToken("testOwner", []string{"user"}),
		resource,
		model.QueryListCommons{
			Limit:    3,
			Offset:   0,
			Rights:   "r",
			SortBy:   "name",
			SortDesc: false,
		},
		filter)

	if err != nil {
		t.Error(err)
		return
	}
	if len(result) != 1 {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result, expected) {
		j, _ := json.Marshal(result)
		t.Error(string(j))
		return
	}
}

func TestDeviceGroupAttributes(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, q, w, err := getTestEnv(ctx, wg, t)
	if err != nil {
		fmt.Println(err)
		return
	}

	resource := "device-groups"
	_, err = q.GetClient().DeleteByQuery(resource).Query(elastic.NewMatchAllQuery()).Do(context.Background())
	if err != nil {
		t.Error(err)
		return
	}
	_, err = q.GetClient().Flush().Index(resource).Do(context.Background())
	if err != nil {
		t.Error(err)
		return
	}

	msg, cmd, err := getDeviceGroupTestObj("g1", map[string]interface{}{
		"name":       "g1_name",
		"image":      "g1_image",
		"device_ids": []string{"d1", "d2"},
		"foo":        "bar",
		"bar":        42,
		"attributes": []map[string]interface{}{
			{
				"key":    "a1",
				"value":  "v1",
				"origin": "test",
			},
			{
				"key":   "a2",
				"value": "v2",
			},
		},
		"criteria": []interface{}{
			map[string]interface{}{
				"function_id":     "f1",
				"aspect_id":       "a1",
				"device_class_id": "",
				"interaction":     "request",
			},
			map[string]interface{}{
				"function_id": "f1",
				"aspect_id":   "a2",
				"interaction": "event",
			},
			map[string]interface{}{
				"function_id":     "f2",
				"aspect_id":       "",
				"device_class_id": "dc1",
				"interaction":     "request",
			},
			map[string]interface{}{
				"function_id":     "f3",
				"device_class_id": "dc1",
			},
		},
		"criteria_short": []interface{}{
			"f1_a1__request",
			"f1_a2__event",
			"f2__dc1_request",
			"f3__dc1_",
		},
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, msg, cmd)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	e, _, err := q.GetResourceEntry(resource, "g1")
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(e.Features["name"], "g1_name") {
		t.Error(e)
		return
	}

	expected := []map[string]interface{}{{
		"id":    "g1",
		"name":  "g1_name",
		"image": "g1_image",
		"attributes": []interface{}{
			map[string]interface{}{
				"key":    "a1",
				"value":  "v1",
				"origin": "test",
			},
			map[string]interface{}{
				"key":   "a2",
				"value": "v2",
			},
		},
		"attribute_list": []interface{}{
			"a1=v1",
			"a2=v2",
		},
		"device_ids": []interface{}{"d1", "d2"},
		"criteria": []interface{}{
			map[string]interface{}{
				"function_id":     "f1",
				"aspect_id":       "a1",
				"device_class_id": "",
				"interaction":     "request",
			},
			map[string]interface{}{
				"function_id": "f1",
				"aspect_id":   "a2",
				"interaction": "event",
			},
			map[string]interface{}{
				"function_id":     "f2",
				"aspect_id":       "",
				"device_class_id": "dc1",
				"interaction":     "request",
			},
			map[string]interface{}{
				"function_id":     "f3",
				"device_class_id": "dc1",
			},
		},
		"criteria_short": []interface{}{
			"f1_a1__request",
			"f1_a2__event",
			"f2__dc1_request",
			"f3__dc1_",
		},
		"creator": "testOwner",
		"permissions": map[string]bool{
			"a": true,
			"r": true,
			"w": true,
			"x": true,
		},
		"permission_holders": map[string][]string{
			"admin_users":   {"testOwner"},
			"execute_users": {"testOwner"},
			"read_users":    {"testOwner"},
			"write_users":   {"testOwner"},
		},
		"shared": false,
	}}

	result, err := q.GetList(createTestToken("testOwner", []string{"user"}), resource, model.QueryListCommons{
		Limit:    3,
		Offset:   0,
		Rights:   "r",
		SortBy:   "name",
		SortDesc: false,
	})
	if err != nil {
		t.Error(err)
		return
	}
	if len(result) != 1 {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result, expected) {
		j, _ := json.Marshal(result)
		t.Error(string(j))
		return
	}

	filter := model.Selection{
		And: []model.Selection{
			{
				Condition: model.ConditionConfig{
					Feature:   "features.criteria_short",
					Operation: model.QueryEqualOperation,
					Value:     "f1_a1__request",
				},
			},
			{
				Condition: model.ConditionConfig{
					Feature:   "features.criteria_short",
					Operation: model.QueryEqualOperation,
					Value:     "f1_a2__event",
				},
			},
		},
	}
	if err != nil {
		t.Error(err)
		return
	}
	result, err = q.GetListWithSelection(
		createTestToken("testOwner", []string{"user"}),
		resource,
		model.QueryListCommons{
			Limit:    3,
			Offset:   0,
			Rights:   "r",
			SortBy:   "name",
			SortDesc: false,
		},
		filter)

	if err != nil {
		t.Error(err)
		return
	}
	if len(result) != 1 {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result, expected) {
		j, _ := json.Marshal(result)
		t.Error(string(j))
		return
	}
}

func getDeviceGroupTestObj(id string, obj map[string]interface{}) (msg []byte, command model.CommandWrapper, err error) {
	text := `{
		"command": "PUT",
		"id": "%s",
		"owner": "testOwner",
		"device_group": %s
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
