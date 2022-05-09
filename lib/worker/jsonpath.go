/*
 * Copyright 2018 InfAI (CC SES)
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

package worker

import (
	"context"
	"encoding/json"
	"github.com/PaesslerAG/gval"
	"github.com/PaesslerAG/jsonpath"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"strings"
	"time"
)

func (this *Worker) MsgToFeatures(kind string, msg []byte) (result map[string]interface{}, err error) {
	result = map[string]interface{}{}
	for _, feature := range this.config.Resources[kind].Features {
		result[feature.Name], err = UseJsonPath(msg, feature)
		if err != nil {
			return
		}
	}
	return
}

func (this *Worker) MsgToAnnotations(kind string, annotationTopic string, msg []byte) (result map[string]interface{}, err error) {
	result = map[string]interface{}{}
	for _, feature := range this.config.Resources[kind].Annotations[annotationTopic] {
		result[feature.Name], err = UseJsonPath(msg, feature)
		if err != nil {
			return
		}
	}
	return
}

func UseJsonPath(msg []byte, feature configuration.Feature) (result interface{}, err error) {
	if len(feature.FirstOf) > 0 {
		for _, path := range feature.FirstOf {
			result, err = UseJsonPathWithScript(msg, path)
			if err != nil {
				return result, err
			}
			result = handlePathResultList(result, feature)
			if result != nil {
				return result, err
			}
		}
		return nil, nil
	}
	result, err = UseJsonPathWithScript(msg, feature.Path)
	if err != nil {
		return result, err
	}
	result = handlePathResultList(result, feature)
	return result, err
}

func handlePathResultList(input interface{}, feature configuration.Feature) interface{} {
	if feature.ResultListToFirstElement {
		list, ok := input.([]interface{})
		if !ok {
			return input
		}
		if len(list) == 0 {
			return nil
		}
		return list[0]
	}
	return input
}

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
	if strings.HasSuffix(path, "+") {
		path = path[:len(path)-1]
	}
	v := interface{}(nil)
	err := json.Unmarshal(msg, &v)
	if err != nil {
		return nil, err
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
