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
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestTotal(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config, _, w, err := getTestEnvWithApi(ctx, wg, t)
	if err != nil {
		fmt.Println(err)
		return
	}

	resource := "devices"

	t.Run("create devices", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			dId := "d" + strconv.Itoa(i)
			t.Run("create device "+dId, saveTestDevice(w, resource, dId, map[string]interface{}{"name": "firstsearch", "device_type_id": "dt1"}))
		}

		for i := 5; i < 12; i++ {
			dId := "d" + strconv.Itoa(i)
			t.Run("create device "+dId, saveTestDevice(w, resource, dId, map[string]interface{}{"name": "secondsearch", "device_type_id": "dt2"}))
		}
	})

	time.Sleep(2 * time.Second)

	check := func(query string, expectedSize int) func(t *testing.T) {
		return func(t *testing.T) {
			req, err := http.NewRequest("GET", "http://localhost:"+config.ServerPort+"/v3/total/devices?"+query, nil)
			if err != nil {
				t.Error(err)
				return
			}
			req.Header.Set("Authorization", testtoken)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Error(err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				temp, _ := ioutil.ReadAll(resp.Body)
				t.Error(resp.StatusCode, string(temp))
				return
			}
			result := int(0)
			err = json.NewDecoder(resp.Body).Decode(&result)
			if err != nil {
				t.Error(err)
				return
			}
			t.Log(result)
			if result != expectedSize {
				t.Error(result)
			}
		}
	}

	t.Run("check all", check("", 12))
	t.Run("check search", check("search=firstsearch", 5))
	t.Run("check filter", check("filter=device_type_id:dt2", 7))
}
