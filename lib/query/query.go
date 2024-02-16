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
	"github.com/SENERGY-Platform/permission-search/lib/opensearchclient"
	"github.com/SENERGY-Platform/permission-search/lib/query/modifier"
	"github.com/opensearch-project/opensearch-go"
	"github.com/opensearch-project/opensearch-go/opensearchapi"
	"github.com/opensearch-project/opensearch-go/opensearchutil"
	"log"
	"net/http"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"encoding/json"

	"errors"
)

type Query struct {
	config           configuration.Config
	opensearchClient *opensearch.Client
	timeout          time.Duration
	modifier         *modifier.Modifier
}

func New(config configuration.Config) (result *Query, err error) {
	timeout, err := time.ParseDuration(config.Timeout)
	if err != nil {
		log.Println("ERROR: unable to parse config.Timeout", err)
		return result, err
	}
	client, err := opensearchclient.New(config)
	if err != nil {
		return result, err
	}
	result = &Query{
		config:           config,
		opensearchClient: client,
		timeout:          timeout,
	}
	result.modifier = modifier.New(config, result)
	return result, err
}

func (this *Query) GetClient() *opensearch.Client {
	return this.opensearchClient
}

func (this *Query) getTimeout() (ctx context.Context) {
	ctx, _ = context.WithTimeout(context.Background(), this.timeout)
	return ctx
}

func (this *Query) ResourceExists(kind string, resource string) (exists bool, err error) {
	return this.resourceExists(this.getTimeout(), kind, resource)
}

func (this *Query) resourceExists(ctx context.Context, kind string, resource string) (exists bool, err error) {
	resp, err := this.opensearchClient.Exists(kind, resource, this.opensearchClient.Exists.WithContext(ctx))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("db: got HTTP code %d when it should have been either 200 or 404", resp.StatusCode)
	}
}

func getRightsQuery(rights string, user string, groups []string) (result []map[string]interface{}) {
	if rights == "" {
		rights = "r"
	}
	for _, right := range rights {
		switch right {
		case 'a':
			or := []map[string]interface{}{}
			if user != "" {
				or = append(or, map[string]interface{}{
					"term": map[string]interface{}{
						"admin_users": user,
					},
				})
			}
			if len(groups) > 0 {
				or = append(or, map[string]interface{}{
					"terms": map[string]interface{}{
						"admin_groups": groups,
					},
				})
			}
			result = append(result, map[string]interface{}{
				"bool": map[string]interface{}{
					"should": or,
				},
			})
		case 'r':
			or := []map[string]interface{}{}
			if user != "" {
				or = append(or, map[string]interface{}{
					"term": map[string]interface{}{
						"read_users": user,
					},
				})
			}
			if len(groups) > 0 {
				or = append(or, map[string]interface{}{
					"terms": map[string]interface{}{
						"read_groups": groups,
					},
				})
			}
			result = append(result, map[string]interface{}{
				"bool": map[string]interface{}{
					"should": or,
				},
			})
		case 'w':
			or := []map[string]interface{}{}
			if user != "" {
				or = append(or, map[string]interface{}{
					"term": map[string]interface{}{
						"write_users": user,
					},
				})
			}
			if len(groups) > 0 {
				or = append(or, map[string]interface{}{
					"terms": map[string]interface{}{
						"write_groups": groups,
					},
				})
			}
			result = append(result, map[string]interface{}{
				"bool": map[string]interface{}{
					"should": or,
				},
			})
		case 'x':
			or := []map[string]interface{}{}
			if user != "" {
				or = append(or, map[string]interface{}{
					"term": map[string]interface{}{
						"execute_users": user,
					},
				})
			}
			if len(groups) > 0 {
				or = append(or, map[string]interface{}{
					"terms": map[string]interface{}{
						"execute_groups": groups,
					},
				})
			}
			result = append(result, map[string]interface{}{
				"bool": map[string]interface{}{
					"should": or,
				},
			})
		}
	}
	return
}

func (this *Query) GetRightsToAdministrate(kind string, user string, groups []string) (result []model.ResourceRights, err error) {
	ctx := this.getTimeout()
	resp, err := this.opensearchClient.Search(
		this.opensearchClient.Search.WithIndex(kind),
		this.opensearchClient.Search.WithContext(ctx),
		this.opensearchClient.Search.WithVersion(true),
		this.opensearchClient.Search.WithBody(opensearchutil.NewJSONReader(map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"filter": getRightsQuery("a", user, groups),
				},
			},
		})),
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
	filter := getRightsQuery(rights, token.GetUserId(), token.GetRoles())
	filter = append(filter, map[string]interface{}{
		"term": map[string]interface{}{
			"resource": pureId,
		},
	})
	resp, err := this.opensearchClient.Search(this.opensearchClient.Search.WithIndex(kind),
		this.opensearchClient.Search.WithContext(ctx),
		this.opensearchClient.Search.WithVersion(true),
		this.opensearchClient.Search.WithSize(1),
		this.opensearchClient.Search.WithBody(opensearchutil.NewJSONReader(map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"filter": filter,
				},
			},
		})),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.IsError() {
		return errors.New(resp.String())
	}
	pl := model.SearchResult[model.Entry]{}
	err = json.NewDecoder(resp.Body).Decode(&pl)
	if err != nil {
		return err
	}

	if pl.Hits.Total.Value == 0 {
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
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": append(getRightsQuery(rights, token.GetUserId(), token.GetRoles()), map[string]interface{}{
					"terms": map[string]interface{}{
						"resource": terms,
					},
				}),
			},
		},
	}
	resp, err := this.opensearchClient.Search(
		this.opensearchClient.Search.WithContext(ctx),
		this.opensearchClient.Search.WithIndex(kind),
		this.opensearchClient.Search.WithSize(len(pureIds)),
		this.opensearchClient.Search.WithBody(opensearchutil.NewJSONReader(query)),
	)
	if err != nil {
		return allowed, err
	}
	defer resp.Body.Close()
	if resp.IsError() {
		return allowed, errors.New(resp.String())
	}
	pl := model.SearchResult[model.Entry]{}
	err = json.NewDecoder(resp.Body).Decode(&pl)
	if err != nil {
		return allowed, err
	}
	for _, hit := range pl.Hits.Hits {
		entry := hit.Source
		for _, info := range preparedModify[entry.Resource] {
			allowed[info.RawId] = true
		}
	}
	return allowed, nil
}

func (this *Query) getListFromIds(token auth.Token, kind string, ids []string, queryCommons model.QueryListCommons) (result []map[string]interface{}, total int64, err error) {
	ctx := this.getTimeout()
	terms := []interface{}{}

	pureIds, preparedModify := this.modifier.PrepareListModify(ids)

	for _, id := range pureIds {
		terms = append(terms, id)
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": append(getRightsQuery(queryCommons.Rights, token.GetUserId(), token.GetRoles()), map[string]interface{}{
					"terms": map[string]interface{}{
						"resource": terms,
					},
				}),
			},
		},
	}

	options := []func(*opensearchapi.SearchRequest){
		this.opensearchClient.Search.WithContext(ctx),
		this.opensearchClient.Search.WithIndex(kind),
	}
	options = append(options, withPaginationAndBody(this.opensearchClient.Search, query, queryCommons)...)
	if queryCommons.WithTotal {
		options = append(options, this.opensearchClient.Search.WithTrackTotalHits(true))
	}

	resp, err := this.opensearchClient.Search(options...)
	if err != nil {
		return result, 0, err
	}
	defer resp.Body.Close()
	if resp.IsError() {
		return result, 0, errors.New(resp.String())
	}
	pl := model.SearchResult[model.Entry]{}
	err = json.NewDecoder(resp.Body).Decode(&pl)
	if err != nil {
		return result, 0, err
	}
	modifyCache := modifier.NewModifyResourceReferenceCache()
	total = pl.Hits.Total.Value
	for _, hit := range pl.Hits.Hits {
		entry := hit.Source
		modifiedResults, err := this.modifier.UsePreparedModify(preparedModify, entry, kind, modifyCache)
		if err != nil {
			return result, 0, err
		}
		for _, modifiedResult := range modifiedResults {
			result = append(result, getEntryResult(modifiedResult, token.GetUserId(), token.GetRoles()))
		}
	}
	if len(queryCommons.AddIdModifier) > 0 {
		result, err, _ = this.addParsedModifier(token, kind, result, queryCommons.AddIdModifier, queryCommons.Rights, queryCommons.SortBy, queryCommons.SortDesc)
	}
	return result, total, nil
}

func (this *Query) GetListFromIds(token auth.Token, kind string, ids []string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error) {
	result, _, err = this.getListFromIds(token, kind, ids, queryCommons)
	return
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

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": getRightsQuery(rights, user, groups),
			},
		},
	}

	resp, err := this.opensearchClient.Search(
		this.opensearchClient.Search.WithContext(ctx),
		this.opensearchClient.Search.WithIndex(kind),
		this.opensearchClient.Search.WithVersion(true),
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
		result = append(result, getEntryResult(hit.Source, user, groups))
	}
	return
}

func (this *Query) getList(token auth.Token, kind string, queryCommons model.QueryListCommons) (result []map[string]interface{}, total int64, err error) {
	ctx := this.getTimeout()

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": getRightsQuery(queryCommons.Rights, token.GetUserId(), token.GetRoles()),
			},
		},
	}

	options := []func(*opensearchapi.SearchRequest){
		this.opensearchClient.Search.WithContext(ctx),
		this.opensearchClient.Search.WithIndex(kind),
		this.opensearchClient.Search.WithVersion(true),
	}
	options = append(options, withPaginationAndBody(this.opensearchClient.Search, query, queryCommons)...)
	if queryCommons.WithTotal {
		options = append(options, this.opensearchClient.Search.WithTrackTotalHits(true))
	}

	resp, err := this.opensearchClient.Search(options...)
	if err != nil {
		return result, 0, err
	}
	defer resp.Body.Close()
	if resp.IsError() {
		return result, 0, errors.New(resp.String())
	}
	pl := model.SearchResult[model.Entry]{}
	err = json.NewDecoder(resp.Body).Decode(&pl)
	if err != nil {
		return result, 0, err
	}
	total = pl.Hits.Total.Value
	for _, hit := range pl.Hits.Hits {
		result = append(result, getEntryResult(hit.Source, token.GetUserId(), token.GetRoles()))
	}
	if len(queryCommons.AddIdModifier) > 0 {
		result, err, _ = this.addParsedModifier(token, kind, result, queryCommons.AddIdModifier, queryCommons.Rights, queryCommons.SortBy, queryCommons.SortDesc)
	}
	return
}

func (this *Query) GetList(token auth.Token, kind string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error) {
	result, _, err = this.getList(token, kind, queryCommons)
	return
}

func (this *Query) GetListForUser(kind string, user string, rights string) (result []string, err error) {
	ctx := this.getTimeout()
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": getRightsQuery(rights, user, []string{}),
			},
		},
	}
	resp, err := this.opensearchClient.Search(
		this.opensearchClient.Search.WithIndex(kind),
		this.opensearchClient.Search.WithContext(ctx),
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
	for _, hit := range pl.Hits.Hits {
		result = append(result, hit.Source.Resource)
	}
	return
}

func (this *Query) CheckUser(kind string, resource string, user string, rights string) (err error) {
	ctx := this.getTimeout()
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": append(getRightsQuery(rights, user, []string{}), map[string]interface{}{
					"term": map[string]interface{}{
						"resource": resource,
					},
				}),
			},
		},
	}
	resp, err := this.opensearchClient.Search(
		this.opensearchClient.Search.WithIndex(kind),
		this.opensearchClient.Search.WithContext(ctx),
		this.opensearchClient.Search.WithVersion(true),
		this.opensearchClient.Search.WithSize(1),
		this.opensearchClient.Search.WithBody(opensearchutil.NewJSONReader(query)),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.IsError() {
		return errors.New(resp.String())
	}
	pl := model.SearchResult[model.Entry]{}
	err = json.NewDecoder(resp.Body).Decode(&pl)
	if err != nil {
		return err
	}

	if err == nil && pl.Hits.Total.Value == 0 {
		err = errors.New("access denied")
	}
	return
}

func (this *Query) GetListForGroup(kind string, groups []string, rights string) (result []string, err error) {
	ctx := this.getTimeout()
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": getRightsQuery(rights, "", groups),
			},
		},
	}
	resp, err := this.opensearchClient.Search(
		this.opensearchClient.Search.WithIndex(kind),
		this.opensearchClient.Search.WithContext(ctx),
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
	if err != nil {
		return result, err
	}
	for _, hit := range pl.Hits.Hits {
		result = append(result, hit.Source.Resource)
	}
	return
}

func (this *Query) CheckGroups(kind string, resource string, groups []string, rights string) (err error) {
	ctx := this.getTimeout()
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": append(getRightsQuery(rights, "", groups), map[string]interface{}{
					"term": map[string]interface{}{
						"resource": resource,
					},
				}),
			},
		},
	}
	resp, err := this.opensearchClient.Search(
		this.opensearchClient.Search.WithIndex(kind),
		this.opensearchClient.Search.WithContext(ctx),
		this.opensearchClient.Search.WithVersion(true),
		this.opensearchClient.Search.WithSize(1),
		this.opensearchClient.Search.WithBody(opensearchutil.NewJSONReader(query)),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.IsError() {
		return errors.New(resp.String())
	}
	pl := model.SearchResult[model.Entry]{}
	err = json.NewDecoder(resp.Body).Decode(&pl)
	if err != nil {
		return err
	}

	if err == nil && pl.Hits.Total.Value == 0 {
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

	searchOperation, searchConfig := this.getFeatureSearchInfo(query)
	body := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": getRightsQuery("a", user, groups),
				"must":   []map[string]interface{}{{searchOperation: searchConfig}},
			},
		},
	}
	resp, err := this.opensearchClient.Search(
		this.opensearchClient.Search.WithIndex(kind),
		this.opensearchClient.Search.WithContext(ctx),
		this.opensearchClient.Search.WithVersion(true),
		this.opensearchClient.Search.WithSize(limit),
		this.opensearchClient.Search.WithFrom(offset),
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

	for _, hit := range pl.Hits.Hits {
		result = append(result, hit.Source.ToResourceRights())
	}
	return
}

func (this *Query) SearchListAll(kind string, query string, user string, groups []string, rights string) (result []map[string]interface{}, err error) {
	limit := 1000
	offset := 0
	temp := []map[string]interface{}{}
	for ok := true; ok; ok = len(temp) > 0 {
		temp, err = this.searchListAll(kind, query, user, groups, rights, limit, offset)
		if err != nil {
			return result, err
		}
		result = append(result, temp...)
		offset = offset + limit
	}
	return
}

func (this *Query) selectByFeature(token auth.Token, kind string, feature string, value string, queryCommons model.QueryListCommons) (result []map[string]interface{}, total int64, err error) {
	ctx := this.getTimeout()
	if !strings.HasPrefix(feature, "features.") && !strings.HasPrefix(feature, "annotations.") {
		feature = "features." + feature
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": append(getRightsQuery(queryCommons.Rights, token.GetUserId(), token.GetRoles()), map[string]interface{}{
					"term": map[string]interface{}{
						feature: value,
					},
				}),
			},
		},
	}
	options := []func(*opensearchapi.SearchRequest){
		this.opensearchClient.Search.WithContext(ctx),
		this.opensearchClient.Search.WithIndex(kind),
	}
	options = append(options, withPaginationAndBody(this.opensearchClient.Search, query, queryCommons)...)
	if queryCommons.WithTotal {
		options = append(options, this.opensearchClient.Search.WithTrackTotalHits(true))
	}

	resp, err := this.opensearchClient.Search(options...)
	if err != nil {
		return result, 0, err
	}
	defer resp.Body.Close()
	if resp.IsError() {
		return result, 0, errors.New(resp.String())
	}
	pl := model.SearchResult[model.Entry]{}
	err = json.NewDecoder(resp.Body).Decode(&pl)
	if err != nil {
		return result, 0, err
	}

	total = pl.Hits.Total.Value
	for _, hit := range pl.Hits.Hits {
		result = append(result, getEntryResult(hit.Source, token.GetUserId(), token.GetRoles()))
	}
	if len(queryCommons.AddIdModifier) > 0 {
		result, err, _ = this.addParsedModifier(token, kind, result, queryCommons.AddIdModifier, queryCommons.Rights, queryCommons.SortBy, queryCommons.SortDesc)
	}
	return
}

func (this *Query) SelectByFeature(token auth.Token, kind string, feature string, value string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error) {
	result, _, err = this.selectByFeature(token, kind, feature, value, queryCommons)
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
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": append(getRightsQuery(rights, user, groups), map[string]interface{}{
					"term": map[string]interface{}{
						"features." + field: value,
					},
				}),
			},
		},
	}
	resp, err := this.opensearchClient.Search(
		this.opensearchClient.Search.WithIndex(kind),
		this.opensearchClient.Search.WithContext(ctx),
		this.opensearchClient.Search.WithVersion(true),
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
		result = append(result, getEntryResult(hit.Source, user, groups))
	}
	return
}

func (this *Query) getFeatureSearchInfo(query string) (operator string, config map[string]interface{}) {
	search := strings.TrimSpace(query)
	if strings.Contains(search, "*") {
		return "wildcard", map[string]interface{}{
			"feature_search": map[string]interface{}{"case_insensitive": true, "value": search},
		}
	}
	if !this.config.EnableCombinedWildcardFeatureSearch || strings.ContainsAny(search, " -/_:,;([{&%$") {
		return "match", map[string]interface{}{
			"feature_search": map[string]interface{}{"operator": "AND", "query": search},
		}
	}
	return "bool", map[string]interface{}{
		"should": []map[string]interface{}{
			{
				"wildcard": map[string]interface{}{
					"feature_search": map[string]interface{}{"case_insensitive": true, "value": "*" + search + "*"},
				},
			},
			{
				"match": map[string]interface{}{
					"feature_search": map[string]interface{}{"operator": "AND", "query": search},
				},
			},
		},
	}
}

func (this *Query) searchList(token auth.Token, kind string, query string, queryCommons model.QueryListCommons, selection *model.Selection) (result []map[string]interface{}, total int64, err error) {
	filter := getRightsQuery(queryCommons.Rights, token.GetUserId(), token.GetRoles())
	if selection != nil {
		selectionFilter, err := this.GetFilter(token, *selection)
		if err != nil {
			return result, 0, err
		}
		filter = append(filter, selectionFilter)
	}
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

	options := []func(*opensearchapi.SearchRequest){
		this.opensearchClient.Search.WithContext(ctx),
		this.opensearchClient.Search.WithIndex(kind),
		this.opensearchClient.Search.WithVersion(true),
	}
	options = append(options, withPaginationAndBody(this.opensearchClient.Search, body, queryCommons)...)
	if queryCommons.WithTotal {
		options = append(options, this.opensearchClient.Search.WithTrackTotalHits(true))
	}

	resp, err := this.opensearchClient.Search(options...)
	if err != nil {
		return result, 0, err
	}
	defer resp.Body.Close()
	if resp.IsError() {
		return result, 0, errors.New(resp.String())
	}
	pl := model.SearchResult[model.Entry]{}
	err = json.NewDecoder(resp.Body).Decode(&pl)
	if err != nil {
		return result, 0, err
	}
	total = pl.Hits.Total.Value
	for _, hit := range pl.Hits.Hits {
		result = append(result, getEntryResult(hit.Source, token.GetUserId(), token.GetRoles()))
	}
	if len(queryCommons.AddIdModifier) > 0 {
		result, err, _ = this.addParsedModifier(token, kind, result, queryCommons.AddIdModifier, queryCommons.Rights, queryCommons.SortBy, queryCommons.SortDesc)
	}
	return
}

// SearchList does a text search with query on the feature_search index
// the function allows optionally additional filtering with the selection parameter. when unneeded this parameter may be nil.
func (this *Query) SearchList(token auth.Token, kind string, query string, queryCommons model.QueryListCommons, selection *model.Selection) (result []map[string]interface{}, err error) {
	result, _, err = this.searchList(token, kind, query, queryCommons, selection)
	return
}

func (this *Query) searchListAll(kind string, query string, user string, groups []string, rights string, limit int, offset int) (result []map[string]interface{}, err error) {
	ctx := this.getTimeout()

	searchOperation, searchConfig := this.getFeatureSearchInfo(query)
	body := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": getRightsQuery(rights, user, groups),
				"must":   []map[string]interface{}{{searchOperation: searchConfig}},
			},
		},
	}
	resp, err := this.opensearchClient.Search(
		this.opensearchClient.Search.WithIndex(kind),
		this.opensearchClient.Search.WithContext(ctx),
		this.opensearchClient.Search.WithVersion(true),
		this.opensearchClient.Search.WithSize(limit),
		this.opensearchClient.Search.WithFrom(offset),
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

	for _, hit := range pl.Hits.Hits {
		result = append(result, getEntryResult(hit.Source, user, groups))
	}
	return
}

func (this *Query) SearchOrderedList(kind string, query string, user string, groups []string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error) {
	ctx := this.getTimeout()

	searchOperation, searchConfig := this.getFeatureSearchInfo(query)
	body := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": getRightsQuery(queryCommons.Rights, user, groups),
				"must":   []map[string]interface{}{{searchOperation: searchConfig}},
			},
		},
	}

	options := []func(*opensearchapi.SearchRequest){
		this.opensearchClient.Search.WithContext(ctx),
		this.opensearchClient.Search.WithIndex(kind),
		this.opensearchClient.Search.WithVersion(true),
	}
	options = append(options, withPaginationAndBody(this.opensearchClient.Search, body, queryCommons)...)

	resp, err := this.opensearchClient.Search(options...)
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
		result = append(result, getEntryResult(hit.Source, user, groups))
	}
	return
}

func (this *Query) GetResourceEntry(kind string, resource string) (result model.Entry, version model.ResourceVersion, err error) {
	version, err = this.GetResourceInterface(kind, resource, &result)
	return
}

func (this *Query) GetResourceInterface(kind string, resource string, result interface{}) (version model.ResourceVersion, err error) {
	ctx := this.getTimeout()
	resp, err := this.opensearchClient.Get(
		kind,
		resource,
		this.opensearchClient.Get.WithContext(ctx),
	)
	if err != nil {
		return version, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return version, model.ErrNotFound
	}
	if resp.IsError() {
		return version, errors.New(resp.String())
	}
	pl := model.OpenSearchGetResult{}
	err = json.NewDecoder(resp.Body).Decode(&pl)
	if err != nil {
		return version, err
	}
	version.PrimaryTerm = pl.PrimaryTerm
	version.SeqNo = pl.SeqNo
	if pl.Source == nil {
		return version, model.ErrNotFound
	}
	temp, err := json.Marshal(pl.Source)
	if err != nil {
		return version, err
	}
	err = json.Unmarshal(temp, result)
	if err != nil {
		return version, err
	}
	return version, nil
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

func (this *Query) getListWithSelection(token auth.Token, kind string, queryCommons model.QueryListCommons, selection model.Selection) (result []map[string]interface{}, total int64, err error) {
	filter := getRightsQuery(queryCommons.Rights, token.GetUserId(), token.GetRoles())
	selectionFilter, err := this.GetFilter(token, selection)
	if err != nil {
		return result, 0, err
	}
	filter = append(filter, selectionFilter)
	body := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": filter,
			},
		},
	}

	ctx := this.getTimeout()

	options := []func(*opensearchapi.SearchRequest){
		this.opensearchClient.Search.WithContext(ctx),
		this.opensearchClient.Search.WithIndex(kind),
		this.opensearchClient.Search.WithVersion(true),
	}
	options = append(options, withPaginationAndBody(this.opensearchClient.Search, body, queryCommons)...)
	if queryCommons.WithTotal {
		options = append(options, this.opensearchClient.Search.WithTrackTotalHits(true))
	}

	resp, err := this.opensearchClient.Search(options...)
	if err != nil {
		return result, 0, err
	}
	if resp.IsError() {
		return result, 0, errors.New(resp.String())
	}
	pl := model.SearchResult[model.Entry]{}
	err = json.NewDecoder(resp.Body).Decode(&pl)
	if err != nil {
		return result, 0, err
	}
	total = pl.Hits.Total.Value
	for _, hit := range pl.Hits.Hits {
		result = append(result, getEntryResult(hit.Source, token.GetUserId(), token.GetRoles()))
	}
	if len(queryCommons.AddIdModifier) > 0 {
		result, err, _ = this.addParsedModifier(token, kind, result, queryCommons.AddIdModifier, queryCommons.Rights, queryCommons.SortBy, queryCommons.SortDesc)
	}
	return
}

func (this *Query) GetListWithSelection(token auth.Token, kind string, queryCommons model.QueryListCommons, selection model.Selection) (result []map[string]interface{}, err error) {
	result, _, err = this.getListWithSelection(token, kind, queryCommons, selection)
	return
}

type WithTotal struct {
	Total  int64       `json:"total"`
	Result interface{} `json:"result"`
}

func (this *Query) Query(tokenStr string, query model.QueryMessage) (result interface{}, code int, err error) {
	token, err := auth.Parse(tokenStr)
	if err != nil {
		return result, model.GetErrCode(err), err
	}
	if query.Find != nil {
		var total int64
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
				result, total, err = this.getList(
					token,
					query.Resource,
					query.Find.QueryListCommons)
			} else {
				result, total, err = this.getListWithSelection(
					token,
					query.Resource,
					query.Find.QueryListCommons,
					*query.Find.Filter)
			}
		} else {
			result, total, err = this.searchList(
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
		if query.Find.QueryListCommons.WithTotal {
			result = WithTotal{
				Total:  total,
				Result: result,
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
		var total int64
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
		result, total, err = this.getListFromIds(
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
		if query.ListIds.QueryListCommons.WithTotal {
			result = WithTotal{
				Total:  total,
				Result: result,
			}
		}
	}

	if query.TermAggregate != nil {
		result, err = this.getTermAggregation(token, query.Resource, "r", *query.TermAggregate, query.TermAggregateLimit)
	}
	if err != nil && code == 0 {
		code = model.GetErrCode(err)
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

func withPaginationAndBody(search opensearchapi.Search, query map[string]interface{}, queryCommons model.QueryListCommons) (result []func(*opensearchapi.SearchRequest)) {
	defaultSort := "resource"
	s := defaultSort
	if queryCommons.SortBy != "" {
		s = queryCommons.SortBy
	}
	if s != defaultSort && !strings.HasPrefix(s, "features.") && !strings.HasPrefix(s, "annotations.") {
		s = "features." + s
	}
	sorts := []string{}
	result = append(result, search.WithSize(queryCommons.Limit))
	if queryCommons.After == nil {
		result = append(result, search.WithFrom(queryCommons.Offset))
	} else {
		query["search_after"] = []interface{}{queryCommons.After.SortFieldValue, queryCommons.After.Id}
	}
	if queryCommons.SortDesc {
		sorts = append(sorts, s+":desc")
		if s != defaultSort {
			sorts = append(sorts, "resource:desc")
		}
	} else {
		sorts = append(sorts, s+":asc")
		if s != defaultSort {
			sorts = append(sorts, "resource:asc")
		}
	}
	result = append(result, search.WithSort(sorts...))
	result = append(result, search.WithBody(opensearchutil.NewJSONReader(query)))
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
