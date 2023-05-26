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
	"fmt"
	"github.com/SENERGY-Platform/permission-search/lib/auth"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/query/modifier"
	"net/http"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
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
	if err != nil {
		debug.PrintStack()
	}
	return
}

func interfaceSlice(strings []string) (result []interface{}) {
	for _, str := range strings {
		result = append(result, str)
	}
	return
}

func getRightsQuery(rights string, user string, groups []string) (result []elastic.Query) {
	if rights == "" {
		rights = "r"
	}
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

func (this *Query) CheckUserOrGroup(tokenStr string, kind string, resource string, rights string) (err error) {
	token, err := auth.Parse(tokenStr)
	if err != nil {
		return err
	}
	return this.CheckUserOrGroupFromAuthToken(token, kind, resource, rights)
}

func (this *Query) CheckUserOrGroupFromAuthToken(token auth.Token, kind string, resource string, rights string) (err error) {
	pureId, _ := modifier.SplitModifier(resource)
	ctx := this.getTimeout()
	query := elastic.NewBoolQuery().Filter(append(getRightsQuery(rights, token.GetUserId(), token.GetRoles()), elastic.NewTermQuery("resource", pureId))...)
	resp, err := this.client.Search().Index(kind).Version(true).Query(query).Size(1).Do(ctx)
	if err == nil && resp.Hits.TotalHits.Value == 0 {
		err = model.ErrAccessDenied
	}
	return
}

func (this *Query) CheckListUserOrGroup(token auth.Token, kind string, ids []string, rights string) (allowed map[string]bool, err error) {
	allowed = map[string]bool{}
	ctx := this.getTimeout()
	terms := []interface{}{}
	pureIds, preparedModify := this.modifier.PrepareListModify(ids)
	for _, id := range pureIds {
		terms = append(terms, id)
	}
	query := elastic.NewBoolQuery().Filter(append(getRightsQuery(rights, token.GetUserId(), token.GetRoles()), elastic.NewTermsQuery("resource", terms...))...)
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

func (this *Query) GetListFromIds(token auth.Token, kind string, ids []string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error) {
	ctx := this.getTimeout()
	terms := []interface{}{}

	pureIds, preparedModify := this.modifier.PrepareListModify(ids)

	for _, id := range pureIds {
		terms = append(terms, id)
	}
	query := elastic.NewBoolQuery().Filter(append(getRightsQuery(queryCommons.Rights, token.GetUserId(), token.GetRoles()), elastic.NewTermsQuery("resource", terms...))...)
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
			result = append(result, getEntryResult(modifiedResult, token.GetUserId(), token.GetRoles()))
		}
	}
	if len(queryCommons.AddIdModifier) > 0 {
		result, err, _ = this.addParsedModifier(token, kind, result, queryCommons.AddIdModifier, queryCommons.Rights, queryCommons.SortBy, queryCommons.SortDesc)
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

func (this *Query) GetList(token auth.Token, kind string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error) {
	ctx := this.getTimeout()
	query := elastic.NewBoolQuery().Filter(getRightsQuery(queryCommons.Rights, token.GetUserId(), token.GetRoles())...)
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
		result = append(result, getEntryResult(entry, token.GetUserId(), token.GetRoles()))
	}
	if len(queryCommons.AddIdModifier) > 0 {
		result, err, _ = this.addParsedModifier(token, kind, result, queryCommons.AddIdModifier, queryCommons.Rights, queryCommons.SortBy, queryCommons.SortDesc)
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

func (this *Query) GetRights(tokenStr string, kind string, resource string) (result model.ResourceRights, err error) {
	token, err := auth.Parse(tokenStr)
	if err != nil {
		return result, err
	}
	if !token.IsAdmin() {
		if err := this.CheckUserOrGroup(tokenStr, kind, resource, "a"); err != nil {
			return result, err
		}
	}
	pureIds, preparedModify := this.modifier.PrepareListModify([]string{resource})
	if len(pureIds) == 0 {
		debug.PrintStack()
		return result, errors.New("unexpected modifier behavior")
	}
	entry, _, err := this.GetResourceEntry(kind, pureIds[0])
	if err != nil {
		return result, err
	}
	entries, err := this.modifier.UsePreparedModify(preparedModify, entry, kind, nil)
	if err != nil {
		return result, err
	}
	if len(entries) == 0 {
		debug.PrintStack()
		return result, errors.New("unexpected modifier behavior")
	}
	entry = entries[0]
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

func (this *Query) SelectByFeature(token auth.Token, kind string, feature string, value string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error) {
	ctx := this.getTimeout()
	if !strings.HasPrefix(feature, "features.") && !strings.HasPrefix(feature, "annotations.") {
		feature = "features." + feature
	}

	query := elastic.NewBoolQuery().Filter(append(getRightsQuery(queryCommons.Rights, token.GetUserId(), token.GetRoles()), elastic.NewTermQuery(feature, value))...)
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
		result = append(result, getEntryResult(entry, token.GetUserId(), token.GetRoles()))
	}
	if len(queryCommons.AddIdModifier) > 0 {
		result, err, _ = this.addParsedModifier(token, kind, result, queryCommons.AddIdModifier, queryCommons.Rights, queryCommons.SortBy, queryCommons.SortDesc)
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

// SearchList does a text search with query on the feature_search index
// the function allows optionally additional filtering with the selection parameter. when unneeded this parameter may be nil.
func (this *Query) SearchList(token auth.Token, kind string, query string, queryCommons model.QueryListCommons, selection *model.Selection) (result []map[string]interface{}, err error) {
	elastic_query := elastic.NewBoolQuery().Filter(getRightsQuery(queryCommons.Rights, token.GetUserId(), token.GetRoles())...).Must(elastic.NewMatchQuery("feature_search", query).Operator("AND"))
	if selection != nil {
		filter, err := this.GetFilter(token, *selection)
		if err != nil {
			return result, err
		}
		elastic_query = elastic_query.Filter(filter)
	}
	ctx := this.getTimeout()
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
		result = append(result, getEntryResult(entry, token.GetUserId(), token.GetRoles()))
	}
	if len(queryCommons.AddIdModifier) > 0 {
		result, err, _ = this.addParsedModifier(token, kind, result, queryCommons.AddIdModifier, queryCommons.Rights, queryCommons.SortBy, queryCommons.SortDesc)
	}
	return
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

func (this *Query) GetResourceEntry(kind string, resource string) (result model.Entry, version model.ResourceVersion, err error) {
	version, err = this.GetResourceInterface(kind, resource, &result)
	return
}

func (this *Query) GetResourceInterface(kind string, resource string, result interface{}) (version model.ResourceVersion, err error) {
	ctx := this.getTimeout()
	resp, err := this.client.Get().Index(kind).Id(resource).Do(ctx)
	if elasticErr, ok := err.(*elastic.Error); ok {
		if elasticErr.Status == http.StatusNotFound {
			return version, model.ErrNotFound
		}
	}
	if err != nil {
		debug.PrintStack()
		return version, err
	}
	version.PrimaryTerm = *resp.PrimaryTerm
	version.SeqNo = *resp.SeqNo
	if resp.Source == nil {
		return version, model.ErrNotFound
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
func (this *Query) GetListWithSelection(token auth.Token, kind string, queryCommons model.QueryListCommons, selection model.Selection) (result []map[string]interface{}, err error) {
	ctx := this.getTimeout()
	filter, err := this.GetFilter(token, selection)
	if err != nil {
		return result, err
	}
	query := elastic.NewBoolQuery().Filter(getRightsQuery(queryCommons.Rights, token.GetUserId(), token.GetRoles())...).Filter(filter)
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
		result = append(result, getEntryResult(entry, token.GetUserId(), token.GetRoles()))
	}
	if len(queryCommons.AddIdModifier) > 0 {
		result, err, _ = this.addParsedModifier(token, kind, result, queryCommons.AddIdModifier, queryCommons.Rights, queryCommons.SortBy, queryCommons.SortDesc)
	}
	return
}

func (this *Query) Query(tokenStr string, query model.QueryMessage) (result interface{}, code int, err error) {
	token, err := auth.Parse(tokenStr)
	if err != nil {
		return result, model.GetErrCode(err), err
	}
	if query.Find != nil {
		if query.Find.Limit == 0 {
			query.Find.Limit = 100
		}
		if query.Find.Rights == "" {
			query.Find.Rights = "r"
		}
		err = query.Find.QueryListCommons.Validate()
		if err != nil {
			return result, model.GetErrCode(err), err
		}
		if query.Find.Search == "" {
			if query.Find.Filter == nil {
				result, err = this.GetList(
					token,
					query.Resource,
					query.Find.QueryListCommons)
			} else {
				result, err = this.GetListWithSelection(
					token,
					query.Resource,
					query.Find.QueryListCommons,
					*query.Find.Filter)
			}
		} else {
			result, err = this.SearchList(
				token,
				query.Resource,
				query.Find.Search,
				query.Find.QueryListCommons,
				query.Find.Filter)
		}
		if len(query.Find.AddIdModifier) > 0 {
			result, err, code = this.addParsedModifier(token, query.Resource, result.([]map[string]interface{}), query.Find.AddIdModifier, query.Find.Rights, query.Find.SortBy, query.Find.SortDesc)
			if err != nil {
				return
			}
		}
	}

	if query.CheckIds != nil {
		result, err = this.CheckListUserOrGroup(
			token,
			query.Resource,
			query.CheckIds.Ids,
			query.CheckIds.Rights)
	}

	if query.ListIds != nil {
		if query.ListIds.Limit == 0 {
			query.ListIds.Limit = 100
		}
		if query.ListIds.Rights == "" {
			query.ListIds.Rights = "r"
		}
		err = query.ListIds.QueryListCommons.Validate()
		if err != nil {
			return result, model.GetErrCode(err), err
		}
		result, err = this.GetListFromIds(
			token,
			query.Resource,
			query.ListIds.Ids,
			query.ListIds.QueryListCommons)

		if len(query.ListIds.AddIdModifier) > 0 {
			result, err, code = this.addParsedModifier(token, query.Resource, result.([]map[string]interface{}), query.ListIds.AddIdModifier, query.ListIds.Rights, query.ListIds.SortBy, query.ListIds.SortDesc)
			if err != nil {
				return result, code, err
			}
		}
	}

	if query.TermAggregate != nil {
		result, err = this.getTermAggregation(token, query.Resource, "r", *query.TermAggregate, query.TermAggregateLimit)
	}
	return
}

func (this *Query) List(tokenStr string, kind string, options model.ListOptions) (result []map[string]interface{}, err error) {
	token, err := auth.Parse(tokenStr)
	if err != nil {
		return result, err
	}
	mode, err := options.Mode()
	if err != nil {
		return result, err
	}
	err = options.QueryListCommons.Validate()
	if err != nil {
		return result, err
	}
	switch mode {
	case model.ListOptionsModeTextSearch:
		return this.SearchList(token, kind, options.TextSearch, options.QueryListCommons, nil)
	case model.ListOptionsModeSelection:
		//options.Mode() guaranties that options.Selection is not empty; panic otherwise
		return this.SelectByFeature(token, kind, options.Selection.Feature, options.Selection.Value, options.QueryListCommons)
	case model.ListOptionsModeListIds:
		return this.GetListFromIds(token, kind, options.ListIds, options.QueryListCommons)
	default:
		return this.GetList(token, kind, options.QueryListCommons)
	}
}

func (this *Query) addParsedModifier(token auth.Token, resourceKind string, elements []map[string]interface{}, parsedModifier map[string][]string, rights string, sortBy string, sortDesc bool) (result []map[string]interface{}, err error, code int) {
	if len(elements) == 0 {
		return elements, nil, http.StatusOK
	}
	idList := []string{}
	for _, e := range elements {
		id, ok := e["id"].(string)
		if !ok {
			return nil, errors.New("unable to use add_id_modifier: result id is not string"), http.StatusInternalServerError
		}
		pureId, parameter := modifier.SplitModifier(id)

		if err != nil {
			err = fmt.Errorf("invalid add_id_modifier value: %w", err)
			return nil, err, http.StatusBadRequest
		}
		if parameter == nil {
			parameter = map[string][]string{}
		}
		for key, values := range parsedModifier {
			parameter[key] = values
		}
		idList = append(idList, modifier.JoinModifier(pureId, parameter))
	}
	result, err = this.GetListFromIds(token, resourceKind, idList, model.QueryListCommons{
		Limit:    len(idList),
		Offset:   0,
		Rights:   rights,
		SortBy:   sortBy,
		SortDesc: sortDesc,
	})
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}
	return result, nil, http.StatusOK
}

func (this *Query) AddModifier(token auth.Token, resourceKind string, elements []map[string]interface{}, modifierString string, rights string, sortBy string, sortDesc bool) (result []map[string]interface{}, err error, code int) {
	parsedModifier, err := modifier.DecodeModifierParameter(modifierString)
	if err != nil {
		err = fmt.Errorf("invalid add_id_modifier value: %w", err)
		return nil, err, http.StatusBadRequest
	}
	return this.addParsedModifier(token, resourceKind, elements, parsedModifier, rights, sortBy, sortDesc)
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

func setPaginationAndSort(query *elastic.SearchService, queryCommons model.QueryListCommons) (result *elastic.SearchService) {
	defaultSort := "resource"
	s := defaultSort
	if queryCommons.SortBy != "" {
		s = queryCommons.SortBy
	}
	if s != defaultSort && !strings.HasPrefix(s, "features.") && !strings.HasPrefix(s, "annotations.") {
		s = "features." + s
	}
	if queryCommons.After == nil {
		result = query.From(queryCommons.Offset).Size(queryCommons.Limit).Sort(s, !queryCommons.SortDesc)
	} else {
		result = query.SearchAfter(queryCommons.After.SortFieldValue, queryCommons.After.Id).Size(queryCommons.Limit).Sort(s, !queryCommons.SortDesc)
	}
	if s != defaultSort {
		result = result.Sort("resource", !queryCommons.SortDesc)
	}
	return result
}

func contains(list []string, value string) bool {
	for _, element := range list {
		if element == value {
			return true
		}
	}
	return false
}
