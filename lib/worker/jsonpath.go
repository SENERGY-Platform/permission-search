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
	"encoding/json"
	"github.com/PaesslerAG/jsonpath"
	"strings"
)

func (this *Worker) MsgToFeatures(kind string, msg []byte) (result map[string]interface{}, err error) {
	result = map[string]interface{}{}
	for _, feature := range this.config.Resources[kind].Features {
		result[feature.Name], err = UseJsonPath(msg, feature.Path)
		if err != nil {
			return
		}
	}
	return
}

func (this *Worker) MsgToAnnotations(kind string, annotationTopic string, msg []byte) (result map[string]interface{}, err error) {
	result = map[string]interface{}{}
	for _, feature := range this.config.Resources[kind].Annotations[annotationTopic] {
		result[feature.Name], err = UseJsonPath(msg, feature.Path)
		if err != nil {
			return
		}
	}
	return
}

func UseJsonPath(msg []byte, path string) (interface{}, error) {
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
	/*
		if list, ok := temp.([]interface{}); ok && len(list) == 1 {
			return list[0], nil
		}
	*/
	return temp, nil
}
