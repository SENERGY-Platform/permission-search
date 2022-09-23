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
	"github.com/SENERGY-Platform/permission-search/lib/query/modifier"
	"net/http"
	"sort"
	"strconv"
	"time"

	"encoding/json"

	"errors"

	"github.com/olivere/elastic/v7"
)

type Query struct {
	config   configuration.Config
	client   *elastic.Client
	timeout  time.Duration
	modifier *modifier.Modifier
}

func New(config configuration.Config) (result *Query, err error) {
	timeout, err := time.ParseDuration(config.ElasticTimeout)
	if err != nil {
		return result, err
	}
	client, err := CreateElasticClient(config)
	if err != nil {
		return result, err
	}
	result = &Query{
		config:  config,
		client:  client,
		timeout: timeout,
	}
	result.modifier = modifier.New(config, result)
	return result, err
}

func (this *Query) getTimeout() (ctx context.Context) {
	ctx, _ = context.WithTimeout(context.Background(), this.timeout)
	return ctx
}

func (this *Query) ResourceExists(kind string, resource string) (exists bool, err error) {
	return this.resourceExists(this.getTimeout(), kind, resource)
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
	ctx := this.getTimeout()
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
	pureId, _ := modifier.SplitModifier(resource)
	ctx := this.getTimeout()
	query := elastic.NewBoolQuery().Filter(append(getRightsQuery(rights, user, groups), elastic.NewTermQuery("resource", pureId))...)
	resp, err := this.client.Search().Index(kind).Version(true).Query(query).Size(1).Do(ctx)
	if err == nil && resp.Hits.TotalHits.Value == 0 {
		err = errors.New("access denied")
	}
	return
}

func (this *Query) CheckListUserOrGroup(kind string, ids []string, user string, groups []string, rights string) (allowed map[string]bool, err error) {
	allowed = map[string]bool{}
	ctx := this.getTimeout()
	terms := []interface{}{}
	pureIds, preparedModify := this.modifier.PrepareListModify(ids)
	for _, id := range pureIds {
		terms = append(terms, id)
	}
	query := elastic.NewBoolQuery().Filter(append(getRightsQuery(rights, user, groups), elastic.NewTermsQuery("resource", terms...))...)
	resp, err := this.client.Search().Index(kind).Query(query).Size(len(pureIds)).Do(ctx)
	if err != nil {
		return allowed, err
	}
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return allowed, err
		}
		for _, info := range preparedModify[entry.Resource] {
			allowed[info.RawId] = true
		}
	}
	return allowed, nil
}

func (this *Query) GetListFromIds(kind string, ids []string, user string, groups []string, rights string) (result []map[string]interface{}, err error) {
	ctx := this.getTimeout()
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
		result = append(result, getEntryResult(entry, user, groups))
	}
	return result, nil
}

func (this *Query) GetListFromIdsOrdered(kind string, ids []string, user string, groups []string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error) {
	ctx := this.getTimeout()
	terms := []interface{}{}

	pureIds, preparedModify := this.modifier.PrepareListModify(ids)

	for _, id := range pureIds {
		terms = append(terms, id)
	}
	query := elastic.NewBoolQuery().Filter(append(getRightsQuery(queryCommons.Rights, user, groups), elastic.NewTermsQuery("resource", terms...))...)
	resp, err := setPaginationAndSort(this.client.Search().Index(kind).Query(query), queryCommons).Do(ctx)
	if err != nil {
		return result, err
	}
	modifyCache := modifier.NewModifyResourceReferenceCache()
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return result, err
		}
		modifiedResults, err := this.modifier.UsePreparedModify(preparedModify, entry, kind, modifyCache)
		if err != nil {
			return result, err
		}
		for _, modifiedResult := range modifiedResults {
			result = append(result, getEntryResult(modifiedResult, user, groups))
		}
	}
	return result, nil
}

func (this *Query) GetFullListForUserOrGroup(kind string, user string, groups []string, rights string) (result []map[string]interface{}, err error) {
	limit := 1000
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
	ctx := this.getTimeout()
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
		result = append(result, getEntryResult(entry, user, groups))
	}
	return
}

func (this *Query) GetOrderedListForUserOrGroup(kind string, user string, groups []string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error) {
	ctx := this.getTimeout()
	query := elastic.NewBoolQuery().Filter(getRightsQuery(queryCommons.Rights, user, groups)...)
	resp, err := setPaginationAndSort(this.client.Search().Index(kind).Version(true).Query(query), queryCommons).Do(ctx)
	if err != nil {
		return result, err
	}
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return result, err
		}
		result = append(result, getEntryResult(entry, user, groups))
	}
	return
}

func (this *Query) GetListForUser(kind string, user string, rights string) (result []string, err error) {
	ctx := this.getTimeout()
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
	ctx := this.getTimeout()
	query := elastic.NewBoolQuery().Filter(append(getRightsQuery(rights, user, []string{}), elastic.NewTermQuery("resource", resource))...)
	resp, err := this.client.Search().Index(kind).Version(true).Query(query).Size(1).Do(ctx)
	if err == nil && resp.Hits.TotalHits.Value == 0 {
		err = errors.New("access denied")
	}
	return
}

func (this *Query) GetListForGroup(kind string, groups []string, rights string) (result []string, err error) {
	ctx := this.getTimeout()
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
	ctx := this.getTimeout()
	query := elastic.NewBoolQuery().Filter(append(getRightsQuery(rights, "", groups), elastic.NewTermQuery("resource", resource))...)
	resp, err := this.client.Search().Index(kind).Version(true).Query(query).Size(1).Do(ctx)
	if err == nil && resp.Hits.TotalHits.Value == 0 {
		err = errors.New("access denied")
	}
	return
}

func (this *Query) GetResourceRights(kind string, resource string) (result model.ResourceRights, err error) {
	pureId, _ := modifier.SplitModifier(resource)
	entry, _, err := this.GetResourceEntry(kind, pureId)
	if err != nil {
		return result, err
	}
	result = entry.ToResourceRights()
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
	ctx := this.getTimeout()
	elastic_query := elastic.NewBoolQuery().Filter(getRightsQuery("a", user, groups)...).Must(elastic.NewMatchQuery("feature_search", query).Operator("AND"))
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
	limit := 1000
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

func (this *Query) SelectByFieldOrdered(kind string, field string, value string, user string, groups []string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error) {
	ctx := this.getTimeout()
	query := elastic.NewBoolQuery().Filter(append(getRightsQuery(queryCommons.Rights, user, groups), elastic.NewTermQuery("features."+field, value))...)
	resp, err := setPaginationAndSort(this.client.Search().Index(kind).Query(query), queryCommons).Do(ctx)
	if err != nil {
		return result, err
	}
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return result, err
		}
		result = append(result, getEntryResult(entry, user, groups))
	}
	return
}

func (this *Query) SelectByFieldAll(kind string, field string, value string, user string, groups []string, rights string) (result []map[string]interface{}, err error) {
	limit := 1000
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
	ctx := this.getTimeout()
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
		result = append(result, getEntryResult(entry, user, groups))
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
	ctx := this.getTimeout()
	elastic_query := elastic.NewBoolQuery().Filter(getRightsQuery(rights, user, groups)...).Must(elastic.NewMatchQuery("feature_search", query).Operator("AND"))
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
		result = append(result, getEntryResult(entry, user, groups))
	}
	return
}

func (this *Query) SearchOrderedList(kind string, query string, user string, groups []string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error) {
	ctx := this.getTimeout()
	elastic_query := elastic.NewBoolQuery().Filter(getRightsQuery(queryCommons.Rights, user, groups)...).Must(elastic.NewMatchQuery("feature_search", query).Operator("AND"))
	resp, err := setPaginationAndSort(this.client.Search().Index(kind).Version(true).Query(elastic_query), queryCommons).Do(ctx)
	if err != nil {
		return result, err
	}
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return result, err
		}
		result = append(result, getEntryResult(entry, user, groups))
	}
	return
}

var ErrNotFound = errors.New("not found")

func (this *Query) GetResourceEntry(kind string, resource string) (result model.Entry, version model.ResourceVersion, err error) {
	version, err = this.GetResourceInterface(kind, resource, &result)
	return
}

func (this *Query) GetResourceInterface(kind string, resource string, result interface{}) (version model.ResourceVersion, err error) {
	ctx := this.getTimeout()
	resp, err := this.client.Get().Index(kind).Id(resource).Do(ctx)
	if elasticErr, ok := err.(*elastic.Error); ok {
		if elasticErr.Status == http.StatusNotFound {
			return version, ErrNotFound
		}
	}
	if err != nil {
		return version, err
	}
	version.PrimaryTerm = *resp.PrimaryTerm
	version.SeqNo = *resp.SeqNo
	if resp.Source == nil {
		return version, ErrNotFound
	}
	err = json.Unmarshal(resp.Source, result)
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

func (this *Query) SearchOrderedListWithSelection(kind string, query string, user string, groups []string, queryCommons model.QueryListCommons, selection elastic.Query) (result []map[string]interface{}, err error) {
	ctx := this.getTimeout()
	//elastic_query := elastic.NewBoolQuery().Filter(getRightsQuery(queryCommons.Rights, user, groups)...).Must(elastic.NewMatchQuery("feature_search", query)).Filter(selection)
	elastic_query := elastic.NewBoolQuery().Filter(getRightsQuery(queryCommons.Rights, user, groups)...).Must(elastic.NewMatchQuery("feature_search", query).Operator("AND")).Filter(selection)
	resp, err := setPaginationAndSort(this.client.Search().Index(kind).Version(true).Query(elastic_query), queryCommons).Do(ctx)
	if err != nil {
		return result, err
	}
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return result, err
		}
		result = append(result, getEntryResult(entry, user, groups))
	}
	return
}

func (this *Query) GetOrderedListForUserOrGroupWithSelection(kind string, user string, groups []string, queryCommons model.QueryListCommons, selection elastic.Query) (result []map[string]interface{}, err error) {
	ctx := this.getTimeout()
	query := elastic.NewBoolQuery().Filter(getRightsQuery(queryCommons.Rights, user, groups)...).Filter(selection)
	resp, err := setPaginationAndSort(this.client.Search().Index(kind).Version(true).Query(query), queryCommons).Do(ctx)
	if err != nil {
		return result, err
	}
	for _, hit := range resp.Hits.Hits {
		entry := model.Entry{}
		err = json.Unmarshal(hit.Source, &entry)
		if err != nil {
			return result, err
		}
		result = append(result, getEntryResult(entry, user, groups))
	}
	return
}

func getSharedState(reqUser string, entry model.Entry) bool {
	return entry.Creator != reqUser
}

func getEntryResult(entry model.Entry, user string, groups []string) map[string]interface{} {
	result := map[string]interface{}{}
	for key, value := range entry.Features {
		result[key] = value
	}
	result["id"] = entry.Resource
	result["creator"] = entry.Creator
	if len(entry.Annotations) > 0 {
		result["annotations"] = entry.Annotations
	}
	result["permissions"] = getPermissions(entry, user, groups)
	result["shared"] = getSharedState(user, entry)
	sort.Strings(entry.AdminUsers)
	sort.Strings(entry.ReadUsers)
	sort.Strings(entry.WriteUsers)
	sort.Strings(entry.ExecuteUsers)
	if contains(entry.AdminUsers, user) || hasAdminGroup(entry, groups) {
		permissionHolders := map[string][]string{
			"admin_users":   entry.AdminUsers,
			"read_users":    entry.ReadUsers,
			"write_users":   entry.WriteUsers,
			"execute_users": entry.ExecuteUsers,
		}
		result["permission_holders"] = permissionHolders
	}

	return result
}

func hasAdminGroup(entry model.Entry, groups []string) bool {
	for _, group := range groups {
		if contains(entry.AdminGroups, group) {
			return true
		}
	}
	return false
}

func setPaginationAndSort(query *elastic.SearchService, queryCommons model.QueryListCommons) *elastic.SearchService {
	if queryCommons.After == nil {
		return query.From(queryCommons.Offset).Size(queryCommons.Limit).Sort("features."+queryCommons.SortBy, !queryCommons.SortDesc).Sort("resource", !queryCommons.SortDesc)
	} else {
		return query.SearchAfter(queryCommons.After.SortFieldValue, queryCommons.After.Id).Size(queryCommons.Limit).Sort("features."+queryCommons.SortBy, !queryCommons.SortDesc).Sort("resource", !queryCommons.SortDesc)
	}
}

func contains(list []string, value string) bool {
	for _, element := range list {
		if element == value {
			return true
		}
	}
	return false
}
