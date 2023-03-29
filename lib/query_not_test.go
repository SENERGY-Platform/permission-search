//go:build !ci
// +build !ci

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
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/worker"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestQueryNot(t *testing.T) {
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

	t.Run("create dg1", saveTestDeviceGroup(w, resource, "dg1", map[string]interface{}{"name": "find_dg1_name"}))
	t.Run("create dg2", saveTestDeviceGroup(w, resource, "dg2", map[string]interface{}{"name": "dg2_name"}))
	t.Run("create dg3", saveTestDeviceGroup(w, resource, "dg3", map[string]interface{}{"name": "dg3_name"}))
	t.Run("create dg4", saveTestDeviceGroup(w, resource, "dg4", map[string]interface{}{"name": "find_dg4_name"}))
	t.Run("create dg5", saveTestDeviceGroup(w, resource, "dg5", map[string]interface{}{"name": "find_dg5_name"}))
	t.Run("create dg6", saveTestDeviceGroup(w, resource, "dg6", map[string]interface{}{"name": "dg6_name"}))
	t.Run("create dg7", saveTestDeviceGroup(w, resource, "dg7", map[string]interface{}{"name": "dg7_name"}))

	time.Sleep(2 * time.Second)

	filter := model.Selection{Not: &model.Selection{Condition: model.ConditionConfig{
		Feature:   "id",
		Operation: model.QueryAnyValueInFeatureOperation,
		Value:     []string{"dg2", "dg4", "dg5"},
	}}}
	if err != nil {
		t.Error(err)
		return
	}

	result, err := q.GetListWithSelection(
		createTestToken("testOwner", []string{"user"}),
		resource,
		model.QueryListCommons{
			Limit:    100,
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

	t.Run("checks list", func(t *testing.T) {
		if len(result) != 4 {
			t.Error(len(result), result)
			return
		}
		if !reflect.DeepEqual(result[0]["name"], "dg3_name") {
			t.Error(result)
			return
		}
		if !reflect.DeepEqual(result[1]["name"], "dg6_name") {
			t.Error(result)
			return
		}
		if !reflect.DeepEqual(result[2]["name"], "dg7_name") {
			t.Error(result)
			return
		}
		if !reflect.DeepEqual(result[3]["name"], "find_dg1_name") {
			t.Error(result)
			return
		}
	})

	filter = model.Selection{Not: &model.Selection{Condition: model.ConditionConfig{
		Feature:   "id",
		Operation: model.QueryAnyValueInFeatureOperation,
		Value:     []string{"dg1"},
	}}}
	if err != nil {
		t.Error(err)
		return
	}

	result, err = q.SearchList(
		createTestToken("testOwner", []string{"user"}),
		resource,
		"find",
		model.QueryListCommons{
			Limit:    100,
			Offset:   0,
			Rights:   "r",
			SortBy:   "name",
			SortDesc: false,
		},
		&filter)

	if err != nil {
		t.Error(err)
		return
	}

	t.Run("checks search", func(t *testing.T) {
		if len(result) != 2 {
			t.Error(len(result), result)
			return
		}
		if !reflect.DeepEqual(result[0]["name"], "find_dg4_name") {
			t.Error(result)
			return
		}
		if !reflect.DeepEqual(result[1]["name"], "find_dg5_name") {
			t.Error(result)
			return
		}
	})
}

func saveTestDeviceGroup(w *worker.Worker, resource string, id string, fields map[string]interface{}) func(t *testing.T) {
	return func(t *testing.T) {
		msg, cmd, err := getDeviceGroupTestObj(id, fields)
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
