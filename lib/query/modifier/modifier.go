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
	"github.com/SENERGY-Platform/permission-search/lib/model"
)

func New(config configuration.Config, query Query) *Modifier {
	return &Modifier{
		config: config,
		query:  query,
	}
}

type Modifier struct {
	config configuration.Config
	query  Query
}

type Query interface {
	GetResourceInterface(kind string, resource string, result interface{}) (version model.ResourceVersion, err error)
}

func (this *Modifier) UsePreparedModify(preparedModify map[string][]PreparedModifyInfo, entry model.Entry, kind string, cache ModifyResourceReferenceCache) (result []model.Entry, err error) {
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

func (this *Modifier) PrepareListModify(ids []string) (pureIds []string, prepared map[string][]PreparedModifyInfo) {
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

func (this *Modifier) modifyResult(entry model.Entry, kind string, modifier PreparedModifyInfo, cache ModifyResourceReferenceCache) (result model.Entry, err error) {
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

type ModifyResourceReferenceCache = *map[string]map[string]map[string]interface{}

func NewModifyResourceReferenceCache() ModifyResourceReferenceCache {
	return &map[string]map[string]map[string]interface{}{}
}
