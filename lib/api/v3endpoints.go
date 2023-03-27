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
	"github.com/SENERGY-Platform/permission-search/lib/query"
	"github.com/SENERGY-Platform/permission-search/lib/rigthsproducer"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func init() {
	endpoints = append(endpoints, V3Endpoints)
}

func V3Endpoints(router *httprouter.Router, config configuration.Config, q Query, p *rigthsproducer.Producer) bool {

	router.GET("/v3/administrate/rights/:resource/:id", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		resource := ps.ByName("resource")
		id := ps.ByName("id")
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		rights, err := q.GetRights(token, resource, id)
		if err == query.ErrNotFound {
			http.Error(res, err.Error(), http.StatusNotFound)
			return
		}
		if err == query.ErrAccessDenied {
			http.Error(res, err.Error(), http.StatusForbidden)
			return
		}
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(rights)
	})

	router.GET("/v3/resources/:resource", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
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
			result, err = q.SelectByField(token, resource, field, value, queryListCommons)
		case "ids":
			// no more than 10 ids should be send
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

	router.GET("/v3/total/:resource", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		resource := params.ByName("resource")

		search := request.URL.Query().Get("search")
		selection := request.URL.Query().Get("filter")

		right := request.URL.Query().Get("rights")
		if right == "" {
			right = "r"
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

		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		var result interface{}

		switch mode {
		case "search":
			result, err = q.SearchListTotal(token, resource, search, right)
		case "selection":
			selectionParts := strings.Split(selection, ":")
			if len(selectionParts) < 2 {
				http.Error(writer, "the query parameter 'select' expects a value like 'field_name:field_value'", http.StatusBadRequest)
				return
			}
			field := selectionParts[0]
			value := strings.Join(selectionParts[1:], ":")
			result, err = q.SelectByFieldTotal(token, resource, field, value, right)
		default:
			result, err = q.GetListTotalForUserOrGroup(token, resource, right)
		}

		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(writer).Encode(result)
	})

	router.HEAD("/v3/resources/:resource/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
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
			http.Error(writer, "access denied: "+err.Error(), http.StatusForbidden)
			return
		}
		writer.WriteHeader(200)
	})

	router.GET("/v3/resources/:resource/:id/access", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
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

	router.GET("/v3/aggregates/term/:resource/:term", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		resource := params.ByName("resource")
		term := params.ByName("term")
		limit, err := strconv.Atoi(request.URL.Query().Get("limit"))
		if err != nil {
			limit = 100
		}
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		result, err := q.GetTermAggregation(token, resource, "r", term, limit)

		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(writer).Encode(result)
	})

	router.POST("/v3/query", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
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
			log.Println("DEBUG:", auth.GetAuthToken(request), "\n", string(temp))
		}

		result, code, err := q.Query(token, query)

		if err != nil {
			http.Error(writer, err.Error(), code)
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
