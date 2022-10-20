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
	"fmt"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/jsonpath"
	"strings"
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
			result, err = jsonpath.UseJsonPathWithScript(msg, path)
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
	result, err = jsonpath.UseJsonPathWithScript(msg, feature.Path)
	if err != nil {
		return result, err
	}
	if len(feature.ConcatListElementFields) > 0 {
		result, err = concatListElementFields(result, feature.ConcatListElementFields)
		if err != nil {
			return result, err
		}
	}
	result = handlePathResultList(result, feature)
	return result, err
}

func concatListElementFields(list interface{}, fields []string) (interface{}, error) {
	if list == nil {
		return list, nil
	}
	if len(fields) == 0 {
		return list, nil
	}
	if temp, ok := list.([]interface{}); ok {
		result := []interface{}{}
		for _, element := range temp {
			concated, err := concatElementFields(element, fields)
			if err != nil {
				return result, err
			}
			result = append(result, concated)
		}
		return result, nil
	} else {
		return concatElementFields(list, fields)
	}
}

func concatElementFields(element interface{}, fields []string) (interface{}, error) {
	if element == nil {
		return element, nil
	}
	if len(fields) == 0 {
		return element, nil
	}
	result := ""
	for _, field := range fields {
		if strings.HasPrefix(field, "$.") {
			fieldValue, err := jsonpath.UseJsonPathWithScriptOnObj(element, field)
			if err != nil {
				return result, err
			}
			if fieldValueStr, ok := fieldValue.(string); ok {
				result = result + fieldValueStr
			} else {
				result = result + fmt.Sprint(fieldValue)
			}
		} else {
			result = result + field
		}
	}
	return result, nil
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
