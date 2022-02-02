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
	"github.com/olivere/elastic/v7"
)

func (this *Query) SearchListTotal(kind string, query string, user string, groups []string, rights string) (result int64, err error) {
	ctx := context.Background()
	elastic_query := elastic.NewBoolQuery().Filter(getRightsQuery(rights, user, groups)...).Must(elastic.NewMatchQuery("feature_search", query).Operator("AND"))

	resp, err := this.client.Search().Index(kind).Version(true).Query(elastic_query).TrackTotalHits(true).Size(1).Do(ctx)
	if err != nil {
		return result, err
	}
	return resp.Hits.TotalHits.Value, nil
}

func (this *Query) SelectByFieldTotal(kind string, field string, value string, user string, groups []string, rights string) (result int64, err error) {
	ctx := context.Background()
	query := elastic.NewBoolQuery().Filter(append(getRightsQuery(rights, user, groups), elastic.NewTermQuery("features."+field, value))...)
	resp, err := this.client.Search().Index(kind).Query(query).TrackTotalHits(true).Size(1).Do(ctx)
	if err != nil {
		return result, err
	}
	return resp.Hits.TotalHits.Value, nil
}

func (this *Query) GetListTotalForUserOrGroup(kind string, user string, groups []string, rights string) (result int64, err error) {
	ctx := context.Background()
	query := elastic.NewBoolQuery().Filter(getRightsQuery(rights, user, groups)...)
	resp, err := this.client.Search().Index(kind).Version(true).Query(query).TrackTotalHits(true).Size(1).Do(ctx)
	if err != nil {
		return result, err
	}
	return resp.Hits.TotalHits.Value, nil
}