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
	"encoding/json"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	jsonpath_set "github.com/mdaverde/jsonpath"
	"strconv"
	"strings"
)

func (this *Modifier) applyResultModifyTemplate(entry map[string]interface{}, templateModifier configuration.TemplateModifier, templateName string, parameter []string, cache ModifyResourceReferenceCache) (result map[string]interface{}, err error) {
	buf, err := json.Marshal(entry)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(buf, &result)
	if err != nil {
		return result, err
	}
	if !this.checkModifyConditions(result, templateModifier.Conditions) {
		return result, nil
	}
	parameterMap := map[string]string{}
	for i, value := range parameter {
		parameterMap["p"+strconv.Itoa(i)] = value
	}

	placeholders := map[string]interface{}{
		"this":      result,
		"parameter": parameterMap,
		"seperator": Seperator,
	}

	templateModifier.References, err = this.setReferencesPlaceholders(templateName, templateModifier.References, placeholders)
	if err != nil {
		return result, err
	}
	references, err := this.loadReferences(templateModifier.References, cache)
	if err != nil {
		return result, err
	}
	placeholders["references"] = references

	path := strings.TrimPrefix(templateModifier.Path, "$.")
	value, err := useTemplate(templateName, templateModifier.Value, placeholders)
	if err != nil {
		return result, err
	}
	err = jsonpath_set.Set(result, path, value)
	return result, err
}
