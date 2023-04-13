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
	"github.com/olivere/elastic/v7"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDeviceComplexPathAndAttribute(t *testing.T) {
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

	resource := "devices"
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

	type Attribute struct {
		Key    string `json:"key"`
		Value  string `json:"value"`
		Origin string `json:"origin"`
	}

	type Device struct {
		Id           string      `json:"id"`
		LocalId      string      `json:"local_id"`
		Name         string      `json:"name"`
		Attributes   []Attribute `json:"attributes"`
		DeviceTypeId string      `json:"device_type_id"`
	}

	deviceMsg, deviceCmd, err := getDeviceTestObj("device1", Device{
		Id:         "device1",
		Name:       "device1",
		Attributes: nil,
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, deviceMsg, deviceCmd)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	e, _, err := q.GetResourceEntry(resource, "device1")
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(e.Features["name"], "device1") {
		t.Error(e)
		return
	}
	if !reflect.DeepEqual(e.Features["nickname"], nil) {
		t.Error(e)
		return
	}
	if !reflect.DeepEqual(e.Features["display_name"], "device1") {
		t.Error(e)
		return
	}

	deviceMsg, deviceCmd, err = getDeviceTestObj("device1", Device{
		Id:   "device1",
		Name: "device1",
		Attributes: []Attribute{
			{
				Key:   "shared/nickname",
				Value: "foobar",
			},
		},
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, deviceMsg, deviceCmd)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	e, _, err = q.GetResourceEntry(resource, "device1")
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(e.Features["name"], "device1") {
		t.Error(e)
		return
	}
	if !reflect.DeepEqual(e.Features["nickname"], "foobar") {
		t.Error(e.Features["nickname"])
		t.Error(e.Features["display_name"])
		return
	}
	if !reflect.DeepEqual(e.Features["display_name"], "foobar") {
		t.Error(e)
		return
	}
}

func TestDeviceDisplayNameSort(t *testing.T) {
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

	resource := "devices"
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

	type Attribute struct {
		Key    string `json:"key"`
		Value  string `json:"value"`
		Origin string `json:"origin"`
	}

	type Device struct {
		Id           string      `json:"id"`
		LocalId      string      `json:"local_id"`
		Name         string      `json:"name"`
		Attributes   []Attribute `json:"attributes"`
		DeviceTypeId string      `json:"device_type_id"`
	}

	deviceMsg, deviceCmd, err := getDeviceTestObj("1", Device{
		Id:         "1",
		Name:       "z",
		Attributes: nil,
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, deviceMsg, deviceCmd)
	if err != nil {
		t.Error(err)
		return
	}

	deviceMsg, deviceCmd, err = getDeviceTestObj("2", Device{
		Id:         "2",
		Name:       "Z",
		Attributes: nil,
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, deviceMsg, deviceCmd)
	if err != nil {
		t.Error(err)
		return
	}

	deviceMsg, deviceCmd, err = getDeviceTestObj("3", Device{
		Id:         "3",
		Name:       "A",
		Attributes: nil,
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, deviceMsg, deviceCmd)
	if err != nil {
		t.Error(err)
		return
	}

	deviceMsg, deviceCmd, err = getDeviceTestObj("4", Device{
		Id:         "4",
		Name:       "a",
		Attributes: nil,
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, deviceMsg, deviceCmd)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	result, err := q.GetList(createTestToken("testOwner", []string{}), "devices", model.QueryListCommons{
		Limit:    100,
		Offset:   0,
		Rights:   "r",
		SortBy:   "display_name",
		SortDesc: false,
	})
	if err != nil {
		t.Error(err)
		return
	}
	expectedNamesOrderCaseInsensitive := []string{"a", "a", "z", "z"}
	if len(result) != len(expectedNamesOrderCaseInsensitive) {
		t.Error(len(result))
		return
	}
	for i, e := range result {
		if strings.ToLower(e["display_name"].(string)) != expectedNamesOrderCaseInsensitive[i] {
			t.Error(e["display_name"])
			return
		}
	}

	result, err = q.GetList(createTestToken("testOwner", []string{}), "devices", model.QueryListCommons{
		Limit:    100,
		Offset:   0,
		Rights:   "r",
		SortBy:   "display_name",
		SortDesc: true,
	})
	if err != nil {
		t.Error(err)
		return
	}
	expectedNamesOrderCaseInsensitive = []string{"z", "z", "a", "a"}
	if len(result) != len(expectedNamesOrderCaseInsensitive) {
		t.Error(len(result))
		return
	}
	for i, e := range result {
		if strings.ToLower(e["display_name"].(string)) != expectedNamesOrderCaseInsensitive[i] {
			t.Error(e["display_name"])
			return
		}
	}
}

func TestReceiveDevice(t *testing.T) {
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

	resource := "devices"
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

	deviceMsg, deviceCmd, err := getDeviceTestObj("device1", map[string]interface{}{
		"name": "device_name_1",
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, deviceMsg, deviceCmd)
	if err != nil {
		t.Error(err)
		return
	}

	attr2 := []interface{}{
		map[string]interface{}{"key": "manufacturer", "value": "42"},
		map[string]interface{}{"key": "api_key", "value": "nope"},
	}
	deviceMsg, deviceCmd, err = getDeviceTestObj("device2", map[string]interface{}{
		"name":       "device_name_2",
		"attributes": attr2,
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, deviceMsg, deviceCmd)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	e, _, err := q.GetResourceEntry(resource, "device1")
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(e.Features["name"], "device_name_1") {
		t.Error(e)
		return
	}
	if !reflect.DeepEqual(e.Features["attributes"], nil) {
		t.Error(e)
		return
	}

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
	if len(result) != 2 {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[0]["name"], "device_name_1") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[1]["name"], "device_name_2") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[1]["attributes"], attr2) {
		t.Error(e)
		return
	}
	t.Log(result)
}

func TestDeviceWithSpecialCharacterAttribute(t *testing.T) {
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

	resource := "devices"
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

	deviceMsg, deviceCmd, err := getDeviceTestObj("device1", map[string]interface{}{
		"name": "device_name_1",
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, deviceMsg, deviceCmd)
	if err != nil {
		t.Error(err)
		return
	}

	attr2 := []interface{}{
		map[string]interface{}{"key": "device-type/manufacturer", "value": "HEAT_COST_ALLOCATOR"},
	}
	deviceMsg, deviceCmd, err = getDeviceTestObj("device2", map[string]interface{}{
		"name":       "device_name_2",
		"attributes": attr2,
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, deviceMsg, deviceCmd)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	filter := model.Selection{
		And: []model.Selection{
			{
				Condition: model.ConditionConfig{
					Feature:   "features.attributes.key",
					Operation: "==",
					Value:     "device-type/manufacturer",
				},
			},
			{
				Condition: model.ConditionConfig{
					Feature:   "features.attributes.value",
					Operation: "==",
					Value:     "HEAT_COST_ALLOCATOR",
				},
			},
		},
	}
	if err != nil {
		t.Error(err)
		return
	}
	result, err := q.GetListWithSelection(
		createTestToken("testOwner", []string{}),
		"devices",
		model.QueryListCommons{
			Limit:    10,
			Offset:   0,
			Rights:   "r",
			SortBy:   "name",
			SortDesc: false,
		},
		filter)

	if len(result) != 1 {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[0]["name"], "device_name_2") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[0]["attributes"], attr2) {
		t.Error(result)
		return
	}
	t.Log(result)
}
