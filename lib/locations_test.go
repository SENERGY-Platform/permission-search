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
	"github.com/opensearch-project/opensearch-go/opensearchutil"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestReceiveLocation(t *testing.T) {
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

	resource := "locations"
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
	locationMsg, locationCmd, err := getLocationTestObj("location1", map[string]interface{}{
		"name":             "location_name",
		"image":            "image1",
		"device_group_ids": []interface{}{"urn:ses:dg1"},
		"device_ids":       []interface{}{"urn:ses:d1"},
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, locationMsg, locationCmd)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	e, _, err := q.GetResourceEntry(resource, "location1")
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(e.Features["name"], "location_name") {
		t.Error(e)
		return
	}

	result, err := q.GetList(createTestToken("testOwner", []string{}), resource, model.QueryListCommons{
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
	if !reflect.DeepEqual(result[0]["name"], "location_name") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[0]["image"], "image1") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[0]["device_group_ids"], []interface{}{"urn:ses:dg1"}) {
		temp, _ := json.Marshal(result[0])
		t.Error(result, "\n", string(temp))
		return
	}
	if !reflect.DeepEqual(result[0]["device_ids"], []interface{}{"urn:ses:d1"}) {
		temp, _ := json.Marshal(result[0])
		t.Error(result, "\n", string(temp))
		return
	}
}

func getLocationTestObj(id string, obj map[string]interface{}) (msg []byte, command model.CommandWrapper, err error) {
	text := `{
		"command": "PUT",
		"id": "%s",
		"owner": "testOwner",
		"location": %s
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
