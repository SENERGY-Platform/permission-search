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

package modifier

import (
	"bytes"
	"encoding/json"
	"github.com/SENERGY-Platform/permission-search/lib/jsonpath"
	"text/template"
)

func useTemplate(name string, tmpl string, placeholders interface{}) (result string, err error) {
	buff := &bytes.Buffer{}
	t, err := template.New(name).Parse(tmpl)
	if err != nil {
		return result, err
	}
	err = t.Execute(buff, placeholders)
	if err != nil {
		return result, err
	}
	result = buff.String()
	return result, nil
}

func jsonPathGetFirst(v interface{}, path string) (value interface{}, err error) {
	value, err = jsonpath.UseJsonPathWithScriptOnObj(normalize(v), path)
	if err != nil {
		return value, err
	}
	if list, ok := value.([]interface{}); ok {
		if len(list) > 0 {
			value = list[0]
		} else {
			value = nil
		}
	}
	return
}

func normalize(v interface{}) (result interface{}) {
	temp, err := json.Marshal(v)
	if err != nil {
		return v
	}
	err = json.Unmarshal(temp, &result)
	if err != nil {
		return v
	}
	return result
}
