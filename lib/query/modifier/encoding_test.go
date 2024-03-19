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
	"github.com/google/uuid"
	"reflect"
	"testing"
)

func TestEncoding(t *testing.T) {
	expected := map[string][]string{"foo": {"bar", "42", "batz"}}
	encoded := EncodeModifierParameter(expected)
	t.Log(encoded)
	result, err := DecodeModifierParameter(encoded)
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(result, expected) {
		t.Error(result)
	}
}

func TestEncodingOrderPreserved(t *testing.T) {
	list := []string{}
	for i := 0; i < 20; i++ {
		uid, _ := uuid.NewUUID()
		list = append(list, uid.String())
	}
	expected := map[string][]string{"foo": list}
	encoded := EncodeModifierParameter(expected)
	result, err := DecodeModifierParameter(encoded)
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(result, expected) {
		t.Error(result)
	}
}

func TestJsonPathGetFirst(t *testing.T) {
	result, err := jsonPathGetFirst(map[string]interface{}{"features": map[string]interface{}{"service_groups": []interface{}{map[string]interface{}{"key": "c3b8fc33-1899-4595-a6fc-eb85ff24a0de", "name": "Total"}}}},
		"$.features.service_groups[?@.key==\"c3b8fc33-1899-4595-a6fc-eb85ff24a0de\"].name")
	if err != nil {
		t.Error(err)
		return
	}
	if result != "Total" {
		t.Error(result)
	}
	result, err = jsonPathGetFirst(map[string]interface{}{"features": map[string]interface{}{"service_groups": []map[string]interface{}{{"key": "c3b8fc33-1899-4595-a6fc-eb85ff24a0de", "name": "Total"}}}},
		"$.features.service_groups[?@.key==\"c3b8fc33-1899-4595-a6fc-eb85ff24a0de\"].name")
	if err != nil {
		t.Error(err)
		return
	}
	if result != "Total" {
		t.Error(result)
	}
}
