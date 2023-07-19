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
	"fmt"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/worker"
	"github.com/opensearch-project/opensearch-go/opensearchutil"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestFeatureConcat(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	msg, _, err := getDeviceGroupTestObj("g1", map[string]interface{}{
		"attributes": []interface{}{
			map[string]interface{}{
				"key":    "platform/generated",
				"value":  "true",
				"origin": "smart-service",
			},
			map[string]interface{}{
				"key":    "platform/smart_service_task",
				"value":  "foobar",
				"origin": "smart-service",
			},
		},
	})

	result, err := worker.UseJsonPath(msg, configuration.Feature{
		Name:                    "attribute_list",
		Path:                    "$.device_group.attributes",
		ConcatListElementFields: []string{"$.key", "=", "$.value"},
	})
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(result, []interface{}{"platform/generated=true", "platform/smart_service_task=foobar"}) {
		t.Error(result)
		return
	}
}

func TestFeatureConcatWithDeviceGroup(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
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
	_, err = q.GetClient().DeleteByQuery([]string{resource}, opensearchutil.NewJSONReader(map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
	}))
	if err != nil {
		t.Error(err)
		return
	}
	_, err = q.GetClient().Indices.Flush(q.GetClient().Indices.Flush.WithIndex(resource))
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
		},
		"criteria_short": []interface{}{
			"f1_a1__request",
			"f1_a2__event",
			"f2__dc1_request",
			"f3__dc1_",
		},
		"attributes": []interface{}{
			map[string]interface{}{
				"key":    "platform/generated",
				"value":  "true",
				"origin": "smart-service",
			},
			map[string]interface{}{
				"key":    "platform/smart_service_task",
				"value":  "foobar",
				"origin": "smart-service",
			},
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
	if !reflect.DeepEqual(e.Features["attribute_list"], []interface{}{"platform/generated=true", "platform/smart_service_task=foobar"}) {
		t.Error(e)
		return
	}
}

func TestDeviceGroupAttributeListFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
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
	_, err = q.GetClient().DeleteByQuery([]string{resource}, opensearchutil.NewJSONReader(map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
	}))
	if err != nil {
		t.Error(err)
		return
	}
	_, err = q.GetClient().Indices.Flush(q.GetClient().Indices.Flush.WithIndex(resource))
	if err != nil {
		t.Error(err)
		return
	}

	msg, cmd, err := getDeviceGroupTestObj("g1", map[string]interface{}{
		"name":       "g1_name_1",
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
		},
		"criteria_short": []interface{}{
			"f1_a1__request",
			"f1_a2__event",
			"f2__dc1_request",
			"f3__dc1_",
		},
		"attributes": []interface{}{
			map[string]interface{}{
				"key":    "platform/generated",
				"value":  "true",
				"origin": "smart-service",
			},
			map[string]interface{}{
				"key":    "platform/smart_service_task",
				"value":  "foobar",
				"origin": "smart-service",
			},
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

	msg2, cmd2, err := getDeviceGroupTestObj("g2", map[string]interface{}{
		"name":       "g2_name_2",
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
		},
		"criteria_short": []interface{}{
			"f1_a1__request",
			"f1_a2__event",
			"f2__dc1_request",
			"f3__dc1_",
		},
		"attributes": []interface{}{
			map[string]interface{}{
				"key":    "platform/smart_service_task",
				"value":  "foobar",
				"origin": "smart-service",
			},
		},
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, msg2, cmd2)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	filter := model.Selection{
		Condition: model.ConditionConfig{
			Feature:   "features.attribute_list",
			Operation: model.QueryEqualOperation,
			Value:     "platform/generated=true",
		},
	}
	if err != nil {
		t.Error(err)
		return
	}

	result, err := q.SearchList(createTestToken("testOwner", []string{"user"}), resource, "name", model.QueryListCommons{
		Limit:  100,
		Offset: 0,
		Rights: "r",
		SortBy: "name",
	}, &filter)

	if err != nil {
		t.Error(err)
		return
	}

	if len(result) != 1 || result[0]["id"] != "g1" {
		t.Error(result)
		return
	}

	filter = model.Selection{
		Not: &model.Selection{
			Condition: model.ConditionConfig{
				Feature:   "features.attribute_list",
				Operation: model.QueryEqualOperation,
				Value:     "platform/generated=true",
			},
		},
	}
	if err != nil {
		t.Error(err)
		return
	}

	result, err = q.SearchList(createTestToken("testOwner", []string{"user"}), resource, "name", model.QueryListCommons{
		Limit:  100,
		Offset: 0,
		Rights: "r",
		SortBy: "name",
	}, &filter)

	if err != nil {
		t.Error(err)
		return
	}

	if len(result) != 1 || result[0]["id"] != "g2" {
		t.Error(result)
		return
	}

	result, err = q.GetListWithSelection(createTestToken("testOwner", []string{"user"}), resource, model.QueryListCommons{
		Limit:  100,
		Offset: 0,
		Rights: "r",
		SortBy: "name",
	}, filter)

	if err != nil {
		t.Error(err)
		return
	}

	if len(result) != 1 || result[0]["id"] != "g2" {
		t.Error(result)
		return
	}
}
