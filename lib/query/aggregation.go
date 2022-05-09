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
	"errors"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/olivere/elastic/v7"
)

func (this *Query) GetTermAggregation(kind string, user string, groups []string, rights string, field string, limit int) (result []model.TermAggregationResultElement, err error) {
	if limit == 0 {
		limit = 100
	}
	ctx := context.Background()
	query := elastic.NewBoolQuery().Filter(getRightsQuery(rights, user, groups)...)
	aggregate := elastic.NewTermsAggregation().Field(field).Size(limit)
	resp, err := this.client.Search().Index(kind).Version(true).Query(query).Aggregation(field, aggregate).Do(ctx)
	if err != nil {
		return result, err
	}
	termsAggregation, found := resp.Aggregations.Terms(field)
	if !found {
		return nil, errors.New("aggregation result not found in response from elasticsearch")
	}
	for _, bucket := range termsAggregation.Buckets {
		result = append(result, model.TermAggregationResultElement{
			Term:  bucket.Key,
			Count: bucket.DocCount,
		})
	}
	return result, err
}
