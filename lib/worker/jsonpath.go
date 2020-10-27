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

	"github.com/JumboInteractiveLimited/jsonpath"
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

func UseJsonPath(msg []byte, path string) (interface{}, error) {
	temp := []interface{}{}
	paths, err := jsonpath.ParsePaths(path)
	if err != nil {
		return nil, err
	}
	eval, err := jsonpath.EvalPathsInBytes(msg, paths)
	if err != nil {
		return nil, err
	}
	for {
		if element, ok := eval.Next(); ok {
			var val interface{}
			err = json.Unmarshal(element.Value, &val)
			if err != nil {
				return nil, err
			}
			temp = append(temp, val)
		} else {
			break
		}
	}
	if len(temp) > 1 {
		return temp, nil
	}
	if len(temp) == 1 {
		return temp[0], nil
	}
	return nil, nil
}
