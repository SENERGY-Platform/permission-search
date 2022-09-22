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

package query

import (
	"bytes"
	"encoding/json"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/jsonpath"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	jsonpath_set "github.com/mdaverde/jsonpath"
	"log"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"text/template"
)

func SplitModifier(id string) (pureId string, modifier map[string][]string) {
	parts := strings.Split(id, "?")
	pureId = parts[0]
	if len(parts) < 2 {
		return
	}
	var err error
	modifier, err = url.ParseQuery(parts[1])
	if err != nil {
		log.Println("WARNING: unable to parse modifier parts as query --> ignore modifiers")
		modifier = nil
		return
	}
	return
}

func (this *Query) usePreparedModify(preparedModify map[string][]PreparedModifyInfo, entry model.Entry, kind string, cache ModifyResourceReferenceCache) (result []model.Entry, err error) {
	for _, modifier := range preparedModify[entry.Resource] {
		if modifier.Unmodified {
			result = append(result, entry)
		} else {
			temp, err := this.modifyResult(entry, kind, modifier, cache)
			if err != nil {
				return result, err
			}
			result = append(result, temp)
		}
	}
	return result, nil
}

func (this *Query) prepareListModify(ids []string) (pureIds []string, prepared map[string][]PreparedModifyInfo) {
	prepared = map[string][]PreparedModifyInfo{}
	pureIdSeen := map[string]bool{}
	for _, id := range ids {
		pureId, modifiers := SplitModifier(id)
		if !pureIdSeen[pureId] {
			pureIdSeen[pureId] = true
			pureIds = append(pureIds, pureId)
		}
		prepared[pureId] = append(prepared[pureId], PreparedModifyInfo{
			RawId:      id,
			PureId:     pureId,
			Unmodified: modifiers == nil || len(modifiers) == 0,
			Modifier:   modifiers,
		})
	}
	return
}

type PreparedModifyInfo struct {
	RawId      string
	PureId     string
	Unmodified bool
	Modifier   map[string][]string
}

func (this *Query) modifyResult(entry model.Entry, kind string, modifier PreparedModifyInfo, cache ModifyResourceReferenceCache) (result model.Entry, err error) {
	var temp map[string]interface{}
	buf, err := json.Marshal(entry)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(buf, &temp)
	if err != nil {
		return result, err
	}
	temp["resource"] = modifier.RawId
	modifierConfigs := this.config.ResultModifiers[kind]
	for _, modifierConfig := range modifierConfigs {
		modifyParameter, modifierIsUsed := modifier.Modifier[modifierConfig.Modifier]
		if modifierIsUsed {
			if modifierConfig.Template != nil {
				temp, err = this.applyResultModifyTemplate(temp, *modifierConfig.Template, kind+"."+modifierConfig.Modifier, modifyParameter, cache)
			}
		}
	}
	buf, err = json.Marshal(temp)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(buf, &result)
	if err != nil {
		return result, err
	}
	return result, nil
}

func (this *Query) applyResultModifyTemplate(entry map[string]interface{}, templateModifier configuration.TemplateModifier, templateName string, parameter []string, cache ModifyResourceReferenceCache) (result map[string]interface{}, err error) {
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

type ModifyResourceReferenceCache = *map[string]map[string]map[string]interface{}

func NewModifyResourceReferenceCache() ModifyResourceReferenceCache {
	return &map[string]map[string]map[string]interface{}{}
}

func (this *Query) loadReferences(references map[string]configuration.ModifierReference, cache ModifyResourceReferenceCache) (result map[string]interface{}, err error) {
	result = map[string]interface{}{}
	if cache == nil {
		cache = NewModifyResourceReferenceCache()
	}
	for name, reference := range references {
		resourceElement := map[string]interface{}{}
		usedCache := false
		if _, ok := (*cache)[reference.Resource]; !ok {
			(*cache)[reference.Resource] = map[string]map[string]interface{}{}
		}
		if element, ok := (*cache)[reference.Resource][reference.ResourceId]; ok {
			resourceElement = element
			usedCache = true
		}
		if !usedCache {
			_, err = this.getResourceInterface(reference.Resource, reference.ResourceId, &resourceElement)
			if err != nil {
				log.Println("WARNING: getResourceInterface() element not found", "\n\t", reference.Resource, "\n\t", reference.ResourceId, "\n\t", err)
				result[name] = reference.Default
				continue
			}
			(*cache)[reference.Resource][reference.ResourceId] = resourceElement
		}
		value, err := jsonPathGetFirst(resourceElement, reference.Path)
		if err != nil {
			log.Println("WARNING: loadReferences() path error", "\n\t", reference.Path, "\n\t", resourceElement, "\n\t", err)
			result[name] = reference.Default
			continue
		}
		if value != nil {
			log.Println("WARNING: loadReferences() path not found", "\n\t", reference.Path, "\n\t", resourceElement)
			result[name] = value
		} else {
			result[name] = reference.Default
		}
	}
	return result, nil
}

func (this *Query) setReferencesPlaceholders(templateNamePrefix string, references map[string]configuration.ModifierReference, placeholders map[string]interface{}) (result map[string]configuration.ModifierReference, err error) {
	result = map[string]configuration.ModifierReference{}
	for key, reference := range references {
		result[key], err = this.setReferencePlaceholders(templateNamePrefix+".referenceTmpl."+key, reference, placeholders)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func (this *Query) setReferencePlaceholders(templNamePrefix string, reference configuration.ModifierReference, placeholders map[string]interface{}) (result configuration.ModifierReference, err error) {
	result = configuration.ModifierReference{
		Resource:   "",
		ResourceId: "",
		Path:       "",
		Default:    reference.Default,
	}
	result.ResourceId, err = useTemplate(templNamePrefix+".resource_id", reference.ResourceId, placeholders)
	if err != nil {
		return result, err
	}
	result.Resource, err = useTemplate(templNamePrefix+".resource", reference.Resource, placeholders)
	if err != nil {
		return result, err
	}
	result.Path, err = useTemplate(templNamePrefix+".path", reference.Path, placeholders)
	if err != nil {
		return result, err
	}
	return
}

func (this *Query) checkModifyConditions(entry map[string]interface{}, conditions []configuration.ModifierCondition) bool {
	for _, condition := range conditions {
		switch condition.Operator {
		case "not_empty":
			value, err := jsonPathGetFirst(entry, condition.Path)
			if err != nil {
				log.Println("ERROR:", err)
				debug.PrintStack()
				return false
			}
			if value == nil {
				return false
			}
		}
	}
	return true
}

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
	value, err = jsonpath.UseJsonPathWithScriptOnObj(v, path)
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
