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

func TestReceiveCharacteristic(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, q, w, err := getTestEnv(ctx, wg, t)
	if err != nil {
		t.Error(err)
		return
	}

	resource := "characteristics"
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

	characteristicMsg, characteristicCmd, err := getCharacteristicTestObj("characteristic0", map[string]interface{}{
		"name":         "characteristic_name_0",
		"display_unit": "display 0",
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, characteristicMsg, characteristicCmd)
	if err != nil {
		t.Error(err)
		return
	}

	characteristicMsg, characteristicCmd, err = getCharacteristicTestObj("characteristic1", map[string]interface{}{
		"name":         "characteristic_name_1",
		"display_unit": "display 1",
		"min_value":    0,
		"max_value":    100,
		"value":        42,
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, characteristicMsg, characteristicCmd)
	if err != nil {
		t.Error(err)
		return
	}

	characteristicMsg, characteristicCmd, err = getCharacteristicTestObj("characteristic2", map[string]interface{}{
		"name":         "characteristic_name_2",
		"display_unit": "display 2",
		"min_value":    "a",
		"max_value":    "x",
		"value":        "foo",
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, characteristicMsg, characteristicCmd)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	e, _, err := q.GetResourceEntry(resource, "characteristic1")
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(e.Features["name"], "characteristic_name_1") {
		t.Error(e)
		return
	}
	if !reflect.DeepEqual(e.Features["display_unit"], "display 1") {
		t.Error(e)
		return
	}

	if !reflect.DeepEqual(e.Features["raw"], map[string]interface{}{
		"name":         "characteristic_name_1",
		"display_unit": "display 1",
		"min_value":    float64(0),
		"max_value":    float64(100),
		"value":        float64(42),
	}) {
		temp, _ := json.Marshal(e.Features["raw"])
		t.Error(string(temp), "\n", e.Features["raw"])
		return
	}

	result, err := q.GetList(createTestToken("testOwner", []string{"user"}), resource, model.QueryListCommons{
		Limit:    5,
		Offset:   0,
		Rights:   "r",
		SortBy:   "name",
		SortDesc: false,
	})
	if err != nil {
		t.Error(err)
		return
	}
	if len(result) != 3 {
		t.Error(result)
		return
	}

	if !reflect.DeepEqual(result[0]["name"], "characteristic_name_0") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[0]["display_unit"], "display 0") {
		t.Error(result)
		return
	}

	if !reflect.DeepEqual(result[0]["raw"], map[string]interface{}{
		"name":         "characteristic_name_0",
		"display_unit": "display 0",
	}) {
		t.Error(result[0]["raw"])
		return
	}

	if !reflect.DeepEqual(result[1]["name"], "characteristic_name_1") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[1]["display_unit"], "display 1") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[1]["raw"], map[string]interface{}{
		"name":         "characteristic_name_1",
		"display_unit": "display 1",
		"min_value":    float64(0),
		"max_value":    float64(100),
		"value":        float64(42),
	}) {
		t.Error(result[1]["raw"])
		return
	}

	if !reflect.DeepEqual(result[2]["name"], "characteristic_name_2") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[2]["display_unit"], "display 2") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[2]["raw"], map[string]interface{}{
		"name":         "characteristic_name_2",
		"display_unit": "display 2",
		"min_value":    "a",
		"max_value":    "x",
		"value":        "foo",
	}) {
		t.Error(result[2]["raw"])
		return
	}
}

func getCharacteristicTestObj(id string, obj map[string]interface{}) (msg []byte, command model.CommandWrapper, err error) {
	text := `{
		"command": "PUT",
		"id": "%s",
		"owner": "testOwner",
		"characteristic": %s
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
