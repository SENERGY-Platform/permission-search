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

package lib

import (
	"context"
	"encoding/json"
	"log"

	elastic "github.com/olivere/elastic/v7"
)

func UpdateInitialGroupRights() {
	for kind, resourceConfig := range Config.Resources {
		for group, right := range resourceConfig.InitialGroupRights {
			updateInitialResourceGroupRights(kind, group, right)
		}
	}
}

func updateInitialResourceGroupRights(kind string, group string, right string) {
	resources, err := getAllResources(kind)
	if err != nil {
		log.Println("ERROR: unable to find resources to update grouprights; ", kind, group, right, err)
		return
	}
	for _, resource := range resources {
		err = SetGroupRight(kind, resource.Resource, group, right)
		if err != nil {
			log.Println("ERROR: unable to update resources grouprights; ", kind, group, right, err)
			return
		}
	}
}
func getAllResources(kind string) (resources []Entry, err error) {
	log.Fatal("not implemented")
	return
}

func Import(imports map[string][]ResourceRights) (err error) {
	for kind, resources := range imports {
		for _, resource := range resources {
			if err = ImportResource(kind, resource); err != nil {
				return
			}
		}
	}
	return
}

func ImportResource(kind string, resource ResourceRights) (err error) {
	ctx := context.Background()
	entry := Entry{Resource: resource.ResourceId, Features: resource.Features, Creator: resource.Creator}
	entry.SetResourceRights(resource)
	_, err = GetClient().Index().Index(kind).Type(ElasticPermissionType).Id(resource.ResourceId).BodyJson(entry).Do(ctx)
	return
}

func Export() (exports map[string][]ResourceRights, err error) {
	exports = map[string][]ResourceRights{}
	for kind := range Config.Resources {
		exports[kind], err = ExportKindAll(kind)
		if err != nil {
			return
		}
	}
	return
}

func ExportKindAll(kind string) (result []ResourceRights, err error) {
	result = []ResourceRights{}
	limit := 100
	offset := 0
	for {
		temp, err := ExportKind(kind, limit, offset)
		if err != nil {
			return result, err
		}
		result = append(result, temp...)
		if len(temp) < limit {
			return result, err
		}
		offset = offset + limit
	}
	return
}

func ExportKind(kind string, limit int, offset int) (result []ResourceRights, err error) {
	ctx := context.Background()
	query := elastic.NewMatchAllQuery()
	resp, err := GetClient().Search().Index(kind).Type(ElasticPermissionType).Query(query).Size(limit).From(offset).Do(ctx)
	if err != nil {
		return result, err
	}
	for _, hit := range resp.Hits.Hits {
		if hit.Type != ElasticPermissionType {
			log.Println("DEBUG: unknown type", hit.Type)
			continue
		}
		entry := Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return result, err
		}
		result = append(result, entry.ToResourceRights())
	}
	return
}
