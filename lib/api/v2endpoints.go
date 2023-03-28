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
	"encoding/json"
	"github.com/SENERGY-Platform/permission-search/lib/auth"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/rigthsproducer"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"strings"
)

func init() {
	endpoints = append(endpoints, V2Endpoints)
}

func V2Endpoints(router *httprouter.Router, config configuration.Config, q Query, p *rigthsproducer.Producer) bool {
	router.GET("/v2/:resource", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		resource := params.ByName("resource")

		search := request.URL.Query().Get("search")
		selection := request.URL.Query().Get("filter")
		ids := request.URL.Query().Get("ids")

		queryListCommons, err := model.GetQueryListCommonsFromUrlQuery(request.URL.Query())
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		mode := ""
		if search != "" {
			mode = "search"
		}
		if selection != "" {
			if mode != "" {
				http.Error(writer, "the query parameters "+mode+" and 'select' may not be combined", http.StatusBadRequest)
				return
			}
			mode = "selection"
		}
		if ids != "" {
			if mode != "" {
				http.Error(writer, "the query parameters "+mode+" and 'ids' may not be combined", http.StatusBadRequest)
				return
			}
			mode = "ids"
		}

		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		var result []map[string]interface{}

		switch mode {
		case "search":
			result, err = q.SearchList(token, resource, search, queryListCommons, nil)
		case "selection":
			selectionParts := strings.Split(selection, ":")
			if len(selectionParts) < 2 {
				http.Error(writer, "the query parameter 'select' expects a value like 'field_name:field_value'", http.StatusBadRequest)
				return
			}
			field := selectionParts[0]
			value := strings.Join(selectionParts[1:], ":")
			result, err = q.SelectByFeature(token, resource, field, value, queryListCommons)
		case "ids":
			// not more than 10 ids should be send
			result, err = q.GetListFromIds(token, resource, strings.Split(ids, ","), queryListCommons)
		default:
			result, err = q.GetList(token, resource, queryListCommons)
		}

		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(writer).Encode(result)
	})

	router.HEAD("/v2/:resource/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		resource := params.ByName("resource")
		id := params.ByName("id")
		right := request.URL.Query().Get("rights")
		if right == "" {
			right = "r"
		}
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		err = q.CheckUserOrGroup(token, resource, id, right)
		if err != nil {
			http.Error(writer, "access denied: "+err.Error(), http.StatusUnauthorized)
			return
		}
		writer.WriteHeader(200)
	})

	router.GET("/v2/:resource/:id/access", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		resource := params.ByName("resource")
		id := params.ByName("id")
		right := request.URL.Query().Get("rights")
		if right == "" {
			right = "r"
		}
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		err = q.CheckUserOrGroup(token, resource, id, right)
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err != nil {
			json.NewEncoder(writer).Encode(false)
		} else {
			json.NewEncoder(writer).Encode(true)
		}
	})

	router.POST("/v2/query", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		query := model.QueryMessage{}
		err = json.NewDecoder(request.Body).Decode(&query)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		if config.Debug {
			temp, _ := json.Marshal(query)
			log.Println("DEBUG:", string(temp))
		}
		var result interface{}
		if query.Find != nil {
			if query.Find.Limit == 0 {
				query.Find.Limit = 100
			}
			if query.Find.SortBy == "" {
				query.Find.SortBy = "name"
			}
			if query.Find.Rights == "" {
				query.Find.Rights = "r"
			}
			if query.Find.Search == "" {
				if query.Find.Filter == nil {
					result, err = q.GetList(
						token,
						query.Resource,
						query.Find.QueryListCommons)
				} else {
					result, err = q.GetListWithSelection(
						token,
						query.Resource,
						query.Find.QueryListCommons,
						*query.Find.Filter)
				}
			} else {
				result, err = q.SearchList(
					token,
					query.Resource,
					query.Find.Search,
					query.Find.QueryListCommons,
					query.Find.Filter)
			}
		}

		if query.CheckIds != nil {
			result, err = q.CheckListUserOrGroup(
				token,
				query.Resource,
				query.CheckIds.Ids,
				query.CheckIds.Rights)
		}

		if query.ListIds != nil {
			if query.ListIds.Limit == 0 {
				query.ListIds.Limit = 100
			}
			if query.ListIds.SortBy == "" {
				query.ListIds.SortBy = "name"
			}
			if query.ListIds.Rights == "" {
				query.ListIds.Rights = "r"
			}
			result, err = q.GetListFromIds(
				token,
				query.Resource,
				query.ListIds.Ids,
				query.ListIds.QueryListCommons)
		}

		if query.TermAggregate != nil {
			result, err = q.GetTermAggregation(token, query.Resource, "r", *query.TermAggregate, query.TermAggregateLimit)
		}

		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		if config.Debug {
			temp, _ := json.Marshal(result)
			log.Println("DEBUG:", string(temp))
		}

		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(writer).Encode(result)
	})
	return true
}
