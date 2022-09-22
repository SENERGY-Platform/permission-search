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

package jsonpath

import (
	"context"
	"encoding/json"
	"github.com/PaesslerAG/gval"
	"github.com/PaesslerAG/jsonpath"
	"strings"
	"time"
)

func UseJsonPathWithoutScript(msg []byte, path string) (interface{}, error) {
	if strings.HasSuffix(path, "+") {
		path = path[:len(path)-1]
	}
	v := interface{}(nil)
	err := json.Unmarshal(msg, &v)
	if err != nil {
		return nil, err
	}
	temp, err := jsonpath.Get(path, v)
	if err != nil {
		if strings.HasPrefix(err.Error(), "unknown key") {
			err = nil
		}
		return nil, err
	}
	return temp, nil
}

func UseJsonPathWithScript(msg []byte, path string) (interface{}, error) {
	v := interface{}(nil)
	err := json.Unmarshal(msg, &v)
	if err != nil {
		return nil, err
	}
	return UseJsonPathWithScriptOnObj(v, path)
}

func UseJsonPathWithScriptOnObj(v interface{}, path string) (interface{}, error) {
	if strings.HasSuffix(path, "+") {
		path = path[:len(path)-1]
	}
	parser, err := gval.Full(jsonpath.PlaceholderExtension()).NewEvaluable(path)
	if err != nil {
		return nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	temp, err := parser(ctx, v)
	if err != nil {
		if strings.HasPrefix(err.Error(), "unknown key") {
			err = nil
		}
		return nil, err
	}
	return temp, nil
}
