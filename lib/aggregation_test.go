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
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestTermAggregation(t *testing.T) {
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

	t.Run("create d1", saveTestDevice(w, resource, "d1", map[string]interface{}{"name": "find_d1_name", "device_type_id": "dt1"}))
	t.Run("create d2", saveTestDevice(w, resource, "d2", map[string]interface{}{"name": "d2_name", "device_type_id": "dt1"}))
	t.Run("create d3", saveTestDevice(w, resource, "d3", map[string]interface{}{"name": "d3_name", "device_type_id": "dt2"}))
	t.Run("create d4", saveTestDevice(w, resource, "d4", map[string]interface{}{"name": "find_d4_name", "device_type_id": "dt2"}))
	t.Run("create d5", saveTestDevice(w, resource, "d5", map[string]interface{}{"name": "find_d5_name", "device_type_id": "dt3"}))

	time.Sleep(2 * time.Second)

	t.Run("check device-type aggregation", func(t *testing.T) {
		rights := "r"
		field := "features.device_type_id"
		result, err := q.GetTermAggregation(testtoken, resource, rights, field, 1000)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(result, []model.TermAggregationResultElement{
			{
				Term:  "dt1",
				Count: 2,
			}, {
				Term:  "dt2",
				Count: 2,
			}, {
				Term:  "dt3",
				Count: 1,
			},
		}) {
			temp, _ := json.Marshal(result)
			t.Error(temp)
		}
	})
}

func TestTermAggregationLimit(t *testing.T) {
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

	t.Run("create devices", func(t *testing.T) {
		for i := 0; i < 200; i++ {
			dtId := "dt" + strconv.Itoa(i)
			dId := "d" + strconv.Itoa(i)
			t.Run("create device "+dId, saveTestDevice(w, resource, dId, map[string]interface{}{"name": "find_d1_name", "device_type_id": dtId}))
		}
	})

	time.Sleep(2 * time.Second)

	check := func(limit int, expectedSize int) func(t *testing.T) {
		return func(t *testing.T) {
			rights := "r"
			field := "features.device_type_id"
			result, err := q.GetTermAggregation(testtoken, resource, rights, field, limit)
			if err != nil {
				t.Error(err)
				return
			}
			if len(result) != expectedSize {
				t.Error(len(result))
			}
		}
	}

	t.Run("check limit default", check(0, 100))
	t.Run("check limit 10", check(10, 10))
	t.Run("check limit 100", check(100, 100))
	t.Run("check limit 150", check(150, 150))
}
