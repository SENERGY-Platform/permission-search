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
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"strconv"
	"time"

	"encoding/json"

	"errors"

	elastic "github.com/olivere/elastic/v7"
)

type Query struct {
	config configuration.Config
	client *elastic.Client
}

func New(config configuration.Config) (result *Query, err error) {
	client, err := CreateElasticClient(config)
	return &Query{
		config: config,
		client: client,
	}, err
}

func (this *Query) ResourceExists(kind string, resource string) (exists bool, err error) {
	return this.resourceExists(context.Background(), kind, resource)
}

func (this *Query) resourceExists(context context.Context, kind string, resource string) (exists bool, err error) {
	exists, err = elastic.NewExistsService(this.client).Index(kind).Id(resource).Do(context)
	return
}

func interfaceSlice(strings []string) (result []interface{}) {
	for _, str := range strings {
		result = append(result, str)
	}
	return
}

func getRightsQuery(rights string, user string, groups []string) (result []elastic.Query) {
	for _, right := range rights {
		switch right {
		case 'a':
			or := []elastic.Query{}
			if user != "" {
				or = append(or, elastic.NewTermQuery("admin_users", user))
			}
			if len(groups) > 0 {
				or = append(or, elastic.NewTermsQuery("admin_groups", interfaceSlice(groups)...))
			}
			result = append(result, elastic.NewBoolQuery().Filter(elastic.NewBoolQuery().Should(or...)))
		case 'r':
			or := []elastic.Query{}
			if user != "" {
				or = append(or, elastic.NewTermQuery("read_users", user))
			}
			if len(groups) > 0 {
				or = append(or, elastic.NewTermsQuery("read_groups", interfaceSlice(groups)...))
			}
			result = append(result, elastic.NewBoolQuery().Filter(elastic.NewBoolQuery().Should(or...)))
		case 'w':
			or := []elastic.Query{}
			if user != "" {
				or = append(or, elastic.NewTermQuery("write_users", user))
			}
			if len(groups) > 0 {
				or = append(or, elastic.NewTermsQuery("write_groups", interfaceSlice(groups)...))
			}
			result = append(result, elastic.NewBoolQuery().Filter(elastic.NewBoolQuery().Should(or...)))
		case 'x':
			or := []elastic.Query{}
			if user != "" {
				or = append(or, elastic.NewTermQuery("execute_users", user))
			}
			if len(groups) > 0 {
				or = append(or, elastic.NewTermsQuery("execute_groups", interfaceSlice(groups)...))
			}
			result = append(result, elastic.NewBoolQuery().Filter(elastic.NewBoolQuery().Should(or...)))
		}
	}
	return
}

func (this *Query) GetRightsToAdministrate(kind string, user string, groups []string) (result []model.ResourceRights, err error) {
	ctx := context.Background()
	query := elastic.NewBoolQuery().Filter(getRightsQuery("a", user, groups)...)
	resp, err := this.client.Search().Index(kind).Version(true).Query(query).Do(ctx)
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

func (this *Query) CheckUserOrGroup(kind string, resource string, user string, groups []string, rights string) (err error) {
	ctx := context.Background()
	query := elastic.NewBoolQuery().Filter(append(getRightsQuery(rights, user, groups), elastic.NewTermQuery("resource", resource))...)
	resp, err := this.client.Search().Index(kind).Version(true).Query(query).Size(1).Do(ctx)
	if err == nil && resp.Hits.TotalHits.Value == 0 {
		err = errors.New("access denied")
	}
	return
}

func (this *Query) CheckListUserOrGroup(kind string, ids []string, user string, groups []string, rights string) (allowed map[string]bool, err error) {
	allowed = map[string]bool{}
	ctx := context.Background()
	terms := []interface{}{}
	for _, id := range ids {
		terms = append(terms, id)
	}
	query := elastic.NewBoolQuery().Filter(append(getRightsQuery(rights, user, groups), elastic.NewTermsQuery("resource", terms...))...)
	resp, err := this.client.Search().Index(kind).Query(query).Size(len(ids)).Do(ctx)
	if err != nil {
		return allowed, err
	}
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return allowed, err
		}
		allowed[entry.Resource] = true
	}
	return allowed, nil
}

func (this *Query) GetListFromIds(kind string, ids []string, user string, groups []string, rights string) (result []map[string]interface{}, err error) {
	ctx := context.Background()
	terms := []interface{}{}
	for _, id := range ids {
		terms = append(terms, id)
	}
	query := elastic.NewBoolQuery().Filter(append(getRightsQuery(rights, user, groups), elastic.NewTermsQuery("resource", terms...))...)
	resp, err := this.client.Search().Index(kind).Query(query).Size(len(ids)).Do(ctx)
	if err != nil {
		return result, err
	}
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return result, err
		}
		entry.Features["id"] = entry.Resource
		entry.Features["creator"] = entry.Creator
		entry.Features["permissions"] = getPermissions(entry, user, groups)
		entry.Features["shared"] = getSharedState(user, entry)
		result = append(result, entry.Features)
	}
	return result, nil
}

func (this *Query) GetListFromIdsOrdered(kind string, ids []string, user string, groups []string, rights string, limitStr string, offsetStr string, orderfeature string, asc bool) (result []map[string]interface{}, err error) {
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return result, err
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return result, err
	}
	ctx := context.Background()
	terms := []interface{}{}
	for _, id := range ids {
		terms = append(terms, id)
	}
	query := elastic.NewBoolQuery().Filter(append(getRightsQuery(rights, user, groups), elastic.NewTermsQuery("resource", terms...))...)
	resp, err := this.client.Search().Index(kind).Query(query).Size(limit).From(offset).Sort("features."+orderfeature, asc).Do(ctx)
	if err != nil {
		return result, err
	}
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return result, err
		}
		entry.Features["id"] = entry.Resource
		entry.Features["creator"] = entry.Creator
		entry.Features["permissions"] = getPermissions(entry, user, groups)
		entry.Features["shared"] = getSharedState(user, entry)
		result = append(result, entry.Features)
	}
	return result, nil
}

func (this *Query) GetFullListForUserOrGroup(kind string, user string, groups []string, rights string) (result []map[string]interface{}, err error) {
	limit := 20
	offset := 0
	temp := []map[string]interface{}{}
	for ok := true; ok; ok = len(temp) > 0 {
		temp, err = this.getListForUserOrGroup(kind, user, groups, rights, limit, offset)
		if err != nil {
			return result, err
		}
		result = append(result, temp...)
		offset = offset + limit
	}
	return
}

func (this *Query) GetListForUserOrGroup(kind string, user string, groups []string, rights string, limitStr string, offsetStr string) (result []map[string]interface{}, err error) {
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return result, err
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return result, err
	}
	return this.getListForUserOrGroup(kind, user, groups, rights, limit, offset)
}

func (this *Query) getListForUserOrGroup(kind string, user string, groups []string, rights string, limit int, offset int) (result []map[string]interface{}, err error) {
	ctx := context.Background()
	query := elastic.NewBoolQuery().Filter(getRightsQuery(rights, user, groups)...)
	resp, err := this.client.Search().Index(kind).Version(true).Query(query).Size(limit).From(offset).Do(ctx)
	if err != nil {
		return result, err
	}
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return result, err
		}
		entry.Features["id"] = entry.Resource
		entry.Features["creator"] = entry.Creator
		entry.Features["permissions"] = getPermissions(entry, user, groups)
		entry.Features["shared"] = getSharedState(user, entry)
		result = append(result, entry.Features)
	}
	return
}

func (this *Query) GetOrderedListForUserOrGroup(kind string, user string, groups []string, rights string, limitStr string, offsetStr string, orderfeature string, asc bool) (result []map[string]interface{}, err error) {
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return result, err
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return result, err
	}
	ctx := context.Background()
	query := elastic.NewBoolQuery().Filter(getRightsQuery(rights, user, groups)...)
	resp, err := this.client.Search().Index(kind).Version(true).Query(query).Size(limit).From(offset).Sort("features."+orderfeature, asc).Do(ctx)
	if err != nil {
		return result, err
	}
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return result, err
		}
		entry.Features["id"] = entry.Resource
		entry.Features["creator"] = entry.Creator
		entry.Features["permissions"] = getPermissions(entry, user, groups)
		entry.Features["shared"] = getSharedState(user, entry)
		result = append(result, entry.Features)
	}
	return
}

func (this *Query) GetListForUser(kind string, user string, rights string) (result []string, err error) {
	ctx := context.Background()
	query := elastic.NewBoolQuery().Filter(getRightsQuery(rights, user, []string{})...)
	resp, err := this.client.Search().Index(kind).Version(true).Query(query).Do(ctx)
	if err != nil {
		return result, err
	}
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return result, err
		}
		result = append(result, entry.Resource)
	}
	return
}

func (this *Query) CheckUser(kind string, resource string, user string, rights string) (err error) {
	ctx := context.Background()
	query := elastic.NewBoolQuery().Filter(append(getRightsQuery(rights, user, []string{}), elastic.NewTermQuery("resource", resource))...)
	resp, err := this.client.Search().Index(kind).Version(true).Query(query).Size(1).Do(ctx)
	if err == nil && resp.Hits.TotalHits.Value == 0 {
		err = errors.New("access denied")
	}
	return
}

func (this *Query) GetListForGroup(kind string, groups []string, rights string) (result []string, err error) {
	ctx := context.Background()
	query := elastic.NewBoolQuery().Filter(getRightsQuery(rights, "", groups)...)
	resp, err := this.client.Search().Index(kind).Version(true).Query(query).Do(ctx)
	if err != nil {
		return result, err
	}
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return result, err
		}
		result = append(result, entry.Resource)
	}
	return
}

func (this *Query) CheckGroups(kind string, resource string, groups []string, rights string) (err error) {
	ctx := context.Background()
	query := elastic.NewBoolQuery().Filter(append(getRightsQuery(rights, "", groups), elastic.NewTermQuery("resource", resource))...)
	resp, err := this.client.Search().Index(kind).Version(true).Query(query).Size(1).Do(ctx)
	if err == nil && resp.Hits.TotalHits.Value == 0 {
		err = errors.New("access denied")
	}
	return
}

func (this *Query) GetResource(kind string, resource string) (result []model.ResourceRights, err error) {
	entry, _, err := this.GetResourceEntry(kind, resource)
	if err != nil {
		return result, err
	}
	result = []model.ResourceRights{entry.ToResourceRights()}
	return
}

func (this *Query) SearchRightsToAdministrate(kind string, user string, groups []string, query string, limitStr string, offsetStr string) (result []model.ResourceRights, err error) {
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return result, err
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return result, err
	}
	ctx := context.Background()
	elastic_query := elastic.NewBoolQuery().Filter(getRightsQuery("a", user, groups)...).Must(elastic.NewMatchQuery("feature_search", query))
	resp, err := this.client.Search().Index(kind).Version(true).Query(elastic_query).Size(limit).From(offset).Do(ctx)
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

func (this *Query) SearchListAll(kind string, query string, user string, groups []string, rights string) (result []map[string]interface{}, err error) {
	limit := 20
	offset := 0
	temp := []map[string]interface{}{}
	for ok := true; ok; ok = len(temp) > 0 {
		temp, err = this.searchList(kind, query, user, groups, rights, limit, offset)
		if err != nil {
			return result, err
		}
		result = append(result, temp...)
		offset = offset + limit
	}
	return
}

func (this *Query) SelectByFieldOrdered(kind string, field string, value string, user string, groups []string, rights string, limitStr string, offsetStr string, orderfeature string, asc bool) (result []map[string]interface{}, err error) {
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return result, err
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return result, err
	}
	ctx := context.Background()
	query := elastic.NewBoolQuery().Filter(append(getRightsQuery(rights, user, groups), elastic.NewTermQuery("features."+field, value))...)
	resp, err := this.client.Search().Index(kind).Query(query).From(offset).Size(limit).Sort("features."+orderfeature, asc).Do(ctx)
	if err != nil {
		return result, err
	}
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return result, err
		}
		entry.Features["id"] = entry.Resource
		entry.Features["creator"] = entry.Creator
		entry.Features["permissions"] = getPermissions(entry, user, groups)
		entry.Features["shared"] = getSharedState(user, entry)
		result = append(result, entry.Features)
	}
	return
}

func (this *Query) SelectByFieldAll(kind string, field string, value string, user string, groups []string, rights string) (result []map[string]interface{}, err error) {
	limit := 20
	offset := 0
	temp := []map[string]interface{}{}
	for ok := true; ok; ok = len(temp) > 0 {
		temp, err = this.selectByField(kind, field, value, user, groups, rights, limit, offset)
		if err != nil {
			return result, err
		}
		result = append(result, temp...)
		offset = offset + limit
	}
	return
}

func (this *Query) selectByField(kind string, field string, value string, user string, groups []string, rights string, limit int, offset int) (result []map[string]interface{}, err error) {
	ctx := context.Background()
	query := elastic.NewBoolQuery().Filter(append(getRightsQuery(rights, user, groups), elastic.NewTermQuery("features."+field, value))...)
	resp, err := this.client.Search().Index(kind).Version(true).Query(query).From(offset).Size(limit).Do(ctx)
	if err != nil {
		return result, err
	}
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return result, err
		}
		entry.Features["id"] = entry.Resource
		entry.Features["creator"] = entry.Creator
		entry.Features["permissions"] = getPermissions(entry, user, groups)
		entry.Features["shared"] = getSharedState(user, entry)
		result = append(result, entry.Features)
	}
	return
}

func (this *Query) SearchList(kind string, query string, user string, groups []string, rights string, limitStr string, offsetStr string) (result []map[string]interface{}, err error) {
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return result, err
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return result, err
	}
	return this.searchList(kind, query, user, groups, rights, limit, offset)
}

func (this *Query) searchList(kind string, query string, user string, groups []string, rights string, limit int, offset int) (result []map[string]interface{}, err error) {
	ctx := context.Background()
	elastic_query := elastic.NewBoolQuery().Filter(getRightsQuery(rights, user, groups)...).Must(elastic.NewMatchQuery("feature_search", query))
	resp, err := this.client.Search().Index(kind).Version(true).Query(elastic_query).From(offset).Size(limit).Do(ctx)
	if err != nil {
		return result, err
	}
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return result, err
		}
		entry.Features["id"] = entry.Resource
		entry.Features["creator"] = entry.Creator
		entry.Features["permissions"] = getPermissions(entry, user, groups)
		entry.Features["shared"] = getSharedState(user, entry)
		result = append(result, entry.Features)
	}
	return
}

func (this *Query) SearchOrderedList(kind string, query string, user string, groups []string, rights string, orderFeature string, asc bool, limitStr string, offsetStr string) (result []map[string]interface{}, err error) {
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return result, err
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return result, err
	}
	ctx := context.Background()
	elastic_query := elastic.NewBoolQuery().Filter(getRightsQuery(rights, user, groups)...).Must(elastic.NewMatchQuery("feature_search", query))
	resp, err := this.client.Search().Index(kind).Version(true).Query(elastic_query).From(offset).Size(limit).Sort("features."+orderFeature, asc).Do(ctx)
	if err != nil {
		return result, err
	}
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return result, err
		}
		entry.Features["id"] = entry.Resource
		entry.Features["creator"] = entry.Creator
		entry.Features["permissions"] = getPermissions(entry, user, groups)
		entry.Features["shared"] = getSharedState(user, entry)
		result = append(result, entry.Features)
	}
	return
}

func (this *Query) GetResourceEntry(kind string, resource string) (result model.Entry, version model.ResourceVersion, err error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	resp, err := this.client.Get().Index(kind).Id(resource).Do(ctx)
	if err != nil {
		return result, version, err
	}
	version.PrimaryTerm = *resp.PrimaryTerm
	version.SeqNo = *resp.SeqNo
	err = json.Unmarshal(resp.Source, &result)
	return
}

func anyMatch(aList []string, bList []string) bool {
	for _, a := range aList {
		for _, b := range bList {
			if a == b {
				return true
			}
		}
	}
	return false
}

func getPermissions(entry model.Entry, user string, groups []string) (result map[string]bool) {
	result = map[string]bool{
		"r": anyMatch(entry.ReadUsers, []string{user}) || anyMatch(entry.ReadGroups, groups),
		"w": anyMatch(entry.WriteUsers, []string{user}) || anyMatch(entry.WriteGroups, groups),
		"x": anyMatch(entry.ExecuteUsers, []string{user}) || anyMatch(entry.ExecuteGroups, groups),
		"a": anyMatch(entry.AdminUsers, []string{user}) || anyMatch(entry.AdminGroups, groups),
	}
	return
}

func (this *Query) SearchOrderedListWithSelection(kind string, query string, user string, groups []string, rights string, orderFeature string, asc bool, limitStr string, offsetStr string, selection elastic.Query) (result []map[string]interface{}, err error) {
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return result, err
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return result, err
	}
	ctx := context.Background()
	elastic_query := elastic.NewBoolQuery().Filter(getRightsQuery(rights, user, groups)...).Must(elastic.NewMatchQuery("feature_search", query)).Filter(selection)
	resp, err := this.client.Search().Index(kind).Version(true).Query(elastic_query).From(offset).Size(limit).Sort("features."+orderFeature, asc).Do(ctx)
	if err != nil {
		return result, err
	}
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return result, err
		}
		entry.Features["id"] = entry.Resource
		entry.Features["creator"] = entry.Creator
		entry.Features["permissions"] = getPermissions(entry, user, groups)
		entry.Features["shared"] = getSharedState(user, entry)
		result = append(result, entry.Features)
	}
	return
}

func (this *Query) GetOrderedListForUserOrGroupWithSelection(kind string, user string, groups []string, rights string, limitStr string, offsetStr string, orderfeature string, asc bool, selection elastic.Query) (result []map[string]interface{}, err error) {
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return result, err
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return result, err
	}
	ctx := context.Background()
	query := elastic.NewBoolQuery().Filter(getRightsQuery(rights, user, groups)...).Filter(selection)
	resp, err := this.client.Search().Index(kind).Version(true).Query(query).Size(limit).From(offset).Sort("features."+orderfeature, asc).Do(ctx)
	if err != nil {
		return result, err
	}
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return result, err
		}
		entry.Features["id"] = entry.Resource
		entry.Features["creator"] = entry.Creator
		entry.Features["permissions"] = getPermissions(entry, user, groups)
		entry.Features["shared"] = getSharedState(user, entry)
		result = append(result, entry.Features)
	}
	return
}

func getSharedState(reqUser string, entry model.Entry) bool {
	isShared := false
	isAdmin := false
	for _, user := range entry.AdminUsers {
		if reqUser == user {
			isAdmin = true
		} else {
			isShared = true
		}
		if isAdmin && isShared {
			return true
		}
	}
	if !isAdmin {
		return false
	}
	for _, user := range entry.ReadUsers {
		if reqUser != user {
			isShared = true
		}
		if isAdmin && isShared {
			return true
		}
	}
	for _, user := range entry.WriteUsers {
		if reqUser != user {
			isShared = true
		}
		if isAdmin && isShared {
			return true
		}
	}
	for _, user := range entry.ExecuteUsers {
		if reqUser != user {
			isShared = true
		}
		if isAdmin && isShared {
			return true
		}
	}
	return false
}
