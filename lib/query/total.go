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
	"context"
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/permission-search/lib/auth"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/opensearch-project/opensearch-go/opensearchutil"
	"strings"
)

func (this *Query) Total(tokenStr string, kind string, options model.ListOptions) (result int64, err error) {
	token, err := auth.Parse(tokenStr)
	if err != nil {
		return result, err
	}
	mode, err := options.Mode()
	if err != nil {
		return result, err
	}
	switch mode {
	case model.ListOptionsModeTextSearch:
		return this.SearchListTotal(token, kind, options.TextSearch, options.QueryListCommons.Rights)
	case model.ListOptionsModeSelection:
		//options.Mode() guaranties that options.Selection is not empty; panic otherwise
		return this.SelectByFeatureTotal(token, kind, options.Selection.Feature, options.Selection.Value, options.QueryListCommons.Rights)
	default:
		return this.GetListTotalForUserOrGroup(token, kind, options.QueryListCommons.Rights)
	}
}

func (this *Query) SearchListTotal(token auth.Token, kind string, query string, rights string) (result int64, err error) {
	filter := getRightsQuery(rights, token.GetUserId(), token.GetRoles())
	searchOperation, searchConfig := this.getFeatureSearchInfo(query)
	body := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": filter,
				"must":   []map[string]interface{}{{searchOperation: searchConfig}},
			},
		},
	}

	ctx := this.getTimeout()

	resp, err := this.opensearchClient.Search(
		this.opensearchClient.Search.WithContext(ctx),
		this.opensearchClient.Search.WithIndex(kind),
		this.opensearchClient.Search.WithVersion(true),
		this.opensearchClient.Search.WithTrackTotalHits(true),
		this.opensearchClient.Search.WithSize(1),
		this.opensearchClient.Search.WithBody(opensearchutil.NewJSONReader(body)),
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
	return pl.Hits.Total.Value, nil
}

func (this *Query) SelectByFeatureTotal(token auth.Token, kind string, field string, value string, rights string) (result int64, err error) {
	ctx := context.Background()
	if !strings.HasPrefix(field, "features.") && !strings.HasPrefix(field, "annotations.") {
		field = "features." + field
	}
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": append(getRightsQuery(rights, token.GetUserId(), token.GetRoles()), map[string]interface{}{
					"term": map[string]interface{}{
						field: value,
					},
				}),
			},
		},
	}
	resp, err := this.opensearchClient.Search(
		this.opensearchClient.Search.WithContext(ctx),
		this.opensearchClient.Search.WithIndex(kind),
		this.opensearchClient.Search.WithTrackTotalHits(true),
		this.opensearchClient.Search.WithSize(1),
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
	return pl.Hits.Total.Value, nil
}

func (this *Query) GetListTotalForUserOrGroup(token auth.Token, kind string, rights string) (result int64, err error) {
	ctx := context.Background()
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": getRightsQuery(rights, token.GetUserId(), token.GetRoles()),
			},
		},
	}
	resp, err := this.opensearchClient.Search(
		this.opensearchClient.Search.WithContext(ctx),
		this.opensearchClient.Search.WithIndex(kind),
		this.opensearchClient.Search.WithTrackTotalHits(true),
		this.opensearchClient.Search.WithSize(1),
		this.opensearchClient.Search.WithVersion(true),
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
	return pl.Hits.Total.Value, nil
}
