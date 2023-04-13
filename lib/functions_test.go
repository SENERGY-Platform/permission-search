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

func TestReceiveFunction(t *testing.T) {
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

	resource := "functions"
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
	functionMsg, functionCmd, err := getFunctionTestObj("function1", map[string]interface{}{
		"name":         "function_name",
		"display_name": "display",
		"rdf_type":     "function_type",
		"concept_id":   "concept_id_1",
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, functionMsg, functionCmd)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	e, _, err := q.GetResourceEntry(resource, "function1")
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(e.Features["name"], "function_name") {
		t.Error(e)
		return
	}
	if !reflect.DeepEqual(e.Features["display_name"], "display") {
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
	if len(result) != 1 {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[0]["name"], "function_name") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[0]["display_name"], "display") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[0]["rdf_type"], "function_type") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[0]["concept_id"], "concept_id_1") {
		t.Error(result)
		return
	}
}

func getFunctionTestObj(id string, obj map[string]interface{}) (msg []byte, command model.CommandWrapper, err error) {
	text := `{
		"command": "PUT",
		"id": "%s",
		"owner": "testOwner",
		"function": %s
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
