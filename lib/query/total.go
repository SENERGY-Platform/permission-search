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
	"github.com/SENERGY-Platform/permission-search/lib/auth"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/olivere/elastic/v7"
)

func (this *Query) Total(token auth.Token, kind string, options model.ListOptions) (result int64, err error) {
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
	ctx := context.Background()
	elastic_query := elastic.NewBoolQuery().Filter(getRightsQuery(rights, token.GetUserId(), token.GetRoles())...).Must(elastic.NewMatchQuery("feature_search", query).Operator("AND"))

	resp, err := this.client.Search().Index(kind).Version(true).Query(elastic_query).TrackTotalHits(true).Size(1).Do(ctx)
	if err != nil {
		return result, err
	}
	return resp.Hits.TotalHits.Value, nil
}

func (this *Query) SelectByFeatureTotal(token auth.Token, kind string, field string, value string, rights string) (result int64, err error) {
	ctx := context.Background()
	query := elastic.NewBoolQuery().Filter(append(getRightsQuery(rights, token.GetUserId(), token.GetRoles()), elastic.NewTermQuery("features."+field, value))...)
	resp, err := this.client.Search().Index(kind).Query(query).TrackTotalHits(true).Size(1).Do(ctx)
	if err != nil {
		return result, err
	}
	return resp.Hits.TotalHits.Value, nil
}

func (this *Query) GetListTotalForUserOrGroup(token auth.Token, kind string, rights string) (result int64, err error) {
	ctx := context.Background()
	query := elastic.NewBoolQuery().Filter(getRightsQuery(rights, token.GetUserId(), token.GetRoles())...)
	resp, err := this.client.Search().Index(kind).Version(true).Query(query).TrackTotalHits(true).Size(1).Do(ctx)
	if err != nil {
		return result, err
	}
	return resp.Hits.TotalHits.Value, nil
}
