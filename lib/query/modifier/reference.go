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
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"log"
)

func (this *Modifier) loadReferences(references map[string]configuration.ModifierReference, cache ModifyResourceReferenceCache) (result map[string]interface{}, err error) {
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
			_, err = this.query.GetResourceInterface(reference.Resource, reference.ResourceId, &resourceElement)
			if err != nil {
				log.Println("WARNING: getResourceInterface() element not found", "\n\t", reference.Resource, "\n\t", reference.ResourceId, "\n\t", err)
				result[name] = reference.Default
				continue
			}
			(*cache)[reference.Resource][reference.ResourceId] = resourceElement
		}
		value, err := jsonPathGetFirst(resourceElement, reference.Path)
		if err != nil {
			log.Printf("WARNING: loadReferences() path error path=%v \n\tresource=%#v\n\terr=%v\n", reference.Path, resourceElement, err)
			result[name] = reference.Default
			continue
		}
		if value != nil {
			log.Printf("WARNING: loadReferences() path not found path=%v \n\tresource=%#v\n", reference.Path, resourceElement)
			result[name] = value
		} else {
			result[name] = reference.Default
		}
	}
	return result, nil
}

func (this *Modifier) setReferencesPlaceholders(templateNamePrefix string, references map[string]configuration.ModifierReference, placeholders map[string]interface{}) (result map[string]configuration.ModifierReference, err error) {
	result = map[string]configuration.ModifierReference{}
	for key, reference := range references {
		result[key], err = this.setReferencePlaceholders(templateNamePrefix+".referenceTmpl."+key, reference, placeholders)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func (this *Modifier) setReferencePlaceholders(templNamePrefix string, reference configuration.ModifierReference, placeholders map[string]interface{}) (result configuration.ModifierReference, err error) {
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
