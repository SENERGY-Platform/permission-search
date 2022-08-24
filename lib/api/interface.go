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
	"github.com/olivere/elastic/v7"
)

type Query interface {
	//v1
	ResourceExists(kind string, resource string) (exists bool, err error)
	GetRightsToAdministrate(kind string, user string, groups []string) (result []model.ResourceRights, err error)
	GetListFromIds(kind string, ids []string, user string, groups []string, rights string) (result []map[string]interface{}, err error)
	GetFullListForUserOrGroup(kind string, user string, groups []string, rights string) (result []map[string]interface{}, err error)
	GetListForUserOrGroup(kind string, user string, groups []string, rights string, limitStr string, offsetStr string) (result []map[string]interface{}, err error)
	GetListForUser(kind string, user string, rights string) (result []string, err error)
	CheckUser(kind string, resource string, user string, rights string) (err error)
	GetListForGroup(kind string, groups []string, rights string) (result []string, err error)
	CheckGroups(kind string, resource string, groups []string, rights string) (err error)
	SearchRightsToAdministrate(kind string, user string, groups []string, query string, limitStr string, offsetStr string) (result []model.ResourceRights, err error)
	SearchListAll(kind string, query string, user string, groups []string, rights string) (result []map[string]interface{}, err error)
	SelectByFieldAll(kind string, field string, value string, user string, groups []string, rights string) (result []map[string]interface{}, err error)
	SearchList(kind string, query string, user string, groups []string, rights string, limitStr string, offsetStr string) (result []map[string]interface{}, err error)

	//v3
	CheckUserOrGroup(kind string, resource string, user string, groups []string, rights string) (err error)
	CheckListUserOrGroup(kind string, ids []string, user string, groups []string, rights string) (allowed map[string]bool, err error)
	GetResourceRights(kind string, resource string) (result model.ResourceRights, err error)
	GetOrderedListForUserOrGroup(kind string, user string, groups []string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error)
	SelectByFieldOrdered(kind string, field string, value string, user string, groups []string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error)
	GetListFromIdsOrdered(kind string, ids []string, user string, groups []string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error)
	SearchOrderedList(kind string, query string, user string, groups []string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error)
	SearchOrderedListWithSelection(kind string, query string, user string, groups []string, queryCommons model.QueryListCommons, selection elastic.Query) (result []map[string]interface{}, err error)
	GetOrderedListForUserOrGroupWithSelection(kind string, user string, groups []string, queryCommons model.QueryListCommons, selection elastic.Query) (result []map[string]interface{}, err error)

	//selection
	GetFilter(token auth.Token, selection model.Selection) (result elastic.Query, err error)

	//migration
	Import(imports map[string][]model.ResourceRights) (err error)
	Export() (exports map[string][]model.ResourceRights, err error)

	GetTermAggregation(kind string, user string, groups []string, rights string, field string, limit int) (result []model.TermAggregationResultElement, err error)

	SearchListTotal(resource string, search string, id string, roles []string, right string) (int64, error)
	SelectByFieldTotal(resource string, field string, value string, id string, roles []string, right string) (int64, error)
	GetListTotalForUserOrGroup(resource string, id string, roles []string, right string) (int64, error)
}
