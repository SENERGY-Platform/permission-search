/*
 * Copyright 2023 InfAI (CC SES)
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

package client

import (
	"encoding/json"
	"github.com/SENERGY-Platform/permission-search/lib/auth"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"net/http"
)

func Query[Result any](client Client, token auth.Token, query model.QueryMessage) (result Result, code int, err error) {
	temp, code, err := client.Query(token, query)
	if err != nil {
		return result, code, err
	}
	result, err = jsonCast[Result](temp)
	if err != nil {
		code = http.StatusBadRequest
	}
	return result, code, err
}

func List[Result any](client Client, token auth.Token, kind string, options model.ListOptions) (result Result, err error) {
	temp, err := client.List(token, kind, options)
	if err != nil {
		return result, err
	}
	return jsonCast[Result](temp)
}

func jsonCast[Result any](in interface{}) (result Result, err error) {
	temp, err := json.Marshal(in)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(temp, &result)
	return result, err
}
