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

package api

import (
	"github.com/SENERGY-Platform/permission-search/lib/auth"
	"github.com/SENERGY-Platform/permission-search/lib/model"
)

type Query interface {
	//v1
	ResourceExists(kind string, resource string) (exists bool, err error)
	GetRightsToAdministrate(kind string, user string, groups []string) (result []model.ResourceRights, err error)
	GetFullListForUserOrGroup(kind string, user string, groups []string, rights string) (result []map[string]interface{}, err error)
	GetListForUserOrGroup(kind string, user string, groups []string, rights string, limitStr string, offsetStr string) (result []map[string]interface{}, err error)
	GetListForUser(kind string, user string, rights string) (result []string, err error)
	CheckUser(kind string, resource string, user string, rights string) (err error)
	GetListForGroup(kind string, groups []string, rights string) (result []string, err error)
	CheckGroups(kind string, resource string, groups []string, rights string) (err error)
	SearchRightsToAdministrate(kind string, user string, groups []string, query string, limitStr string, offsetStr string) (result []model.ResourceRights, err error)
	SearchListAll(kind string, query string, user string, groups []string, rights string) (result []map[string]interface{}, err error)
	SelectByFieldAll(kind string, field string, value string, user string, groups []string, rights string) (result []map[string]interface{}, err error)

	// SearchList does a text search with query on the feature_search index
	// the function allows optionally additional filtering with the selection parameter. when unneeded this parameter may be nil.
	SearchList(token auth.Token, kind string, query string, queryCommons model.QueryListCommons, selection *model.Selection) (result []map[string]interface{}, err error)
	GetList(token auth.Token, kind string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error)
	GetListFromIds(token auth.Token, kind string, ids []string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error)
	GetListWithSelection(token auth.Token, kind string, queryCommons model.QueryListCommons, selection model.Selection) (result []map[string]interface{}, err error)
	SelectByFeature(token auth.Token, kind string, feature string, value string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error)

	CheckListUserOrGroup(token auth.Token, kind string, ids []string, rights string) (allowed map[string]bool, err error)

	//v3
	V3

	//migration
	Import(imports map[string][]model.ResourceRights) (err error)
	Export(token string) (exports map[string][]model.ResourceRights, err error)
}

type V3 interface {
	Query(token string, query model.QueryMessage) (result interface{}, code int, err error)
	List(token string, kind string, options model.ListOptions) (result []map[string]interface{}, err error)
	Total(token string, kind string, options model.ListOptions) (result int64, err error)

	CheckUserOrGroup(token string, kind string, resource string, rights string) (err error)

	GetRights(token string, kind string, resource string) (result model.ResourceRights, err error)

	GetTermAggregation(token string, kind string, rights string, field string, limit int) (result []model.TermAggregationResultElement, err error)

	ExportKind(token string, kind string, limit int, offset int) (result []model.ResourceRights, err error)
}
