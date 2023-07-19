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
	"errors"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/opensearch-project/opensearch-go/opensearchutil"
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
	ctx := this.getTimeout()
	entry := model.Entry{Resource: resource.ResourceId, Features: resource.Features, Creator: resource.Creator}
	entry.SetResourceRights(resource.ResourceRightsBase)
	resp, err := this.opensearchClient.Index(
		kind,
		opensearchutil.NewJSONReader(entry),
		this.opensearchClient.Index.WithDocumentID(resource.ResourceId),
		this.opensearchClient.Index.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.IsError() {
		return errors.New(resp.String())
	}
	return nil
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
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
	}
	resp, err := this.opensearchClient.Search(
		this.opensearchClient.Search.WithIndex(kind),
		this.opensearchClient.Search.WithContext(ctx),
		this.opensearchClient.Search.WithSize(limit),
		this.opensearchClient.Search.WithFrom(offset),
		this.opensearchClient.Search.WithBody(opensearchutil.NewJSONReader(query)),
	)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	if resp.IsError() {
		return result, errors.New(resp.String())
	}
	pl := model.SearchResult[model.Entry]{}
	err = json.NewDecoder(resp.Body).Decode(&pl)
	if err != nil {
		return result, err
	}
	for _, hit := range pl.Hits.Hits {
		result = append(result, hit.Source.ToResourceRights())
	}
	return
}
