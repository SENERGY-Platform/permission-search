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

package query

import (
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	elastic "github.com/olivere/elastic/v7"
)

func (this *Query) Import(imports map[string][]model.ResourceRights) (err error) {
	for kind, resources := range imports {
		for _, resource := range resources {
			if err = this.ImportResource(kind, resource); err != nil {
				return
			}
		}
	}
	return
}

func (this *Query) ImportResource(kind string, resource model.ResourceRights) (err error) {
	ctx := context.Background()
	entry := model.Entry{Resource: resource.ResourceId, Features: resource.Features, Creator: resource.Creator}
	entry.SetResourceRights(resource.ResourceRightsBase)
	_, err = this.client.Index().Index(kind).Id(resource.ResourceId).BodyJson(entry).Do(ctx)
	return
}

func (this *Query) Export() (exports map[string][]model.ResourceRights, err error) {
	exports = map[string][]model.ResourceRights{}
	for kind := range this.config.Resources {
		exports[kind], err = this.ExportKindAll(kind)
		if err != nil {
			return
		}
	}
	return
}

func (this *Query) ExportKindAll(kind string) (result []model.ResourceRights, err error) {
	result = []model.ResourceRights{}
	limit := 100
	offset := 0
	for {
		temp, err := this.ExportKind(kind, limit, offset)
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

func (this *Query) ExportKind(kind string, limit int, offset int) (result []model.ResourceRights, err error) {
	ctx := context.Background()
	query := elastic.NewMatchAllQuery()
	resp, err := this.client.Search().Index(kind).Query(query).Size(limit).From(offset).Do(ctx)
	if err != nil {
		return result, err
	}
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return result, err
		}
		result = append(result, entry.ToResourceRights())
	}
	return
}
