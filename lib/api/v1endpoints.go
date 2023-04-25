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
)

func init() {
	endpoints = append(endpoints, V1Endpoints)
}

func V1Endpoints(router *httprouter.Router, config configuration.Config, q Query, p *rigthsproducer.Producer) bool {
	router.GET("/administrate/exists/:resource_kind/:resource", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		resource := ps.ByName("resource")
		exists, err := q.ResourceExists(kind, resource)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(exists)
	})

	router.GET("/administrate/rights/:resource_kind", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		list, err := q.GetRightsToAdministrate(kind, token.GetUserId(), token.GetRoles())
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(list)
	})

	router.GET("/administrate/rights/:resource_kind/get/:resource", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		resource := ps.ByName("resource")
		token := auth.GetAuthToken(r)
		rights, err := q.GetRights(token, kind, resource)
		if err == model.ErrNotFound {
			http.Error(res, "404", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(rights)
	})

	router.GET("/administrate/rights/:resource_kind/query/:query/:limit/:offset", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		query := ps.ByName("query")
		limit := ps.ByName("limit")
		offset := ps.ByName("offset")
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		list, err := q.SearchRightsToAdministrate(kind, token.GetUserId(), token.GetRoles(), query, limit, offset)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(list)
	})

	router.GET("/jwt/search/:resource_kind/:query/:right", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		query := ps.ByName("query")
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		list, err := q.SearchListAll(kind, query, token.GetUserId(), token.GetRoles(), right)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(list)
	})

	router.GET("/jwt/select/:resource_kind/:field/:value/:right", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		field := ps.ByName("field")
		value := ps.ByName("value")
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		list, err := q.SelectByFieldAll(kind, field, value, token.GetUserId(), token.GetRoles(), right)
		if err != nil {
			log.Println("ERROR:", err)
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(list)
	})

	router.GET("/jwt/select/:resource_kind/:field/:value/:right/:limit/:offset/:orderfeature/:direction", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		field := ps.ByName("field")
		value := ps.ByName("value")
		limit := ps.ByName("limit")
		offset := ps.ByName("offset")
		orderfeature := ps.ByName("orderfeature")
		direction := ps.ByName("direction")
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		queryListCommons, err := model.GetQueryListCommonsFromStrings(limit, offset, orderfeature, direction, right)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		list, err := q.SelectByFeature(token, kind, field, value, queryListCommons)
		if err != nil {
			log.Println("ERROR:", err)
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(list)
	})

	router.GET("/jwt/search/:resource_kind/:query/:right/:limit/:offset", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		query := ps.ByName("query")
		limit := ps.ByName("limit")
		offset := ps.ByName("offset")
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		queryCommons, err := model.GetQueryListCommonsFromStrings(limit, offset, "", "", right)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		list, err := q.SearchList(token, kind, query, queryCommons, nil)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(list)
	})

	router.GET("/jwt/search/:resource_kind/:query/:right/:limit/:offset/:orderfeature", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		query := ps.ByName("query")
		limit := ps.ByName("limit")
		offset := ps.ByName("offset")
		order := ps.ByName("orderfeature")
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		queryListCommons, err := model.GetQueryListCommonsFromStrings(limit, offset, order, "asc", right)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		list, err := q.SearchList(token, kind, query, queryListCommons, nil)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(list)
	})

	router.GET("/jwt/search/:resource_kind/:query/:right/:limit/:offset/:orderfeature/asc", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		query := ps.ByName("query")
		limit := ps.ByName("limit")
		offset := ps.ByName("offset")
		order := ps.ByName("orderfeature")
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		queryListCommons, err := model.GetQueryListCommonsFromStrings(limit, offset, order, "asc", right)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		list, err := q.SearchList(token, kind, query, queryListCommons, nil)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(list)
	})

	router.GET("/jwt/search/:resource_kind/:query/:right/:limit/:offset/:orderfeature/desc", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		query := ps.ByName("query")
		limit := ps.ByName("limit")
		offset := ps.ByName("offset")
		order := ps.ByName("orderfeature")
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		queryListCommons, err := model.GetQueryListCommonsFromStrings(limit, offset, order, "desc", right)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		list, err := q.SearchList(token, kind, query, queryListCommons, nil)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(list)
	})

	//TODO: add limit/offset variant
	router.GET("/jwt/list/:resource_kind/:right", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		list, err := q.GetFullListForUserOrGroup(kind, token.GetUserId(), token.GetRoles(), right)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(list)
	})

	router.GET("/jwt/list/:resource_kind/:right/:limit/:offset", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		limit := ps.ByName("limit")
		offset := ps.ByName("offset")
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		list, err := q.GetListForUserOrGroup(kind, token.GetUserId(), token.GetRoles(), right, limit, offset)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(list)
	})

	router.GET("/jwt/list/:resource_kind/:right/:limit/:offset/:orderfeature/asc", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		limit := ps.ByName("limit")
		offset := ps.ByName("offset")
		orderfeature := ps.ByName("orderfeature")
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		queryListCommons, err := model.GetQueryListCommonsFromStrings(limit, offset, orderfeature, "asc", right)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		list, err := q.GetList(token, kind, queryListCommons)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(list)
	})

	router.GET("/jwt/list/:resource_kind/:right/:limit/:offset/:orderfeature/desc", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		limit := ps.ByName("limit")
		offset := ps.ByName("offset")
		orderfeature := ps.ByName("orderfeature")
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		queryListCommons, err := model.GetQueryListCommonsFromStrings(limit, offset, orderfeature, "desc", right)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		list, err := q.GetList(token, kind, queryListCommons)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(list)
	})

	router.GET("/jwt/check/:resource_kind/:resource_id/:right", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		resource := ps.ByName("resource_id")
		token := auth.GetAuthToken(r)
		err := q.CheckUserOrGroup(token, kind, resource, right)
		if err != nil {
			log.Println("access denied", err)
			http.Error(res, "access denied: "+err.Error(), http.StatusUnauthorized)
			return
		}
		ok := map[string]string{"status": "ok"}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(ok)
	})

	router.GET("/jwt/check/:resource_kind/:resource_id/:right/bool", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		resource := ps.ByName("resource_id")
		token := auth.GetAuthToken(r)
		err := q.CheckUserOrGroup(token, kind, resource, right)
		if err != nil {
			res.Header().Set("Content-Type", "application/json; charset=utf-8")
			json.NewEncoder(res).Encode(false)
		} else {
			res.Header().Set("Content-Type", "application/json; charset=utf-8")
			json.NewEncoder(res).Encode(true)
		}
	})

	router.POST("/ids/check/:resource_kind/:right", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		ids := []string{}
		err := json.NewDecoder(r.Body).Decode(&ids)
		if err != nil {
			log.Println("WARNING: error in user send data", err)
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		ok, err := q.CheckListUserOrGroup(token, kind, ids, right)
		if err != nil {
			log.Println("ERROR:", ids, err)
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(ok)
	})

	router.POST("/ids/select/:resource_kind/:right", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		queryListCommons, err := model.GetQueryListCommonsFromUrlQuery(r.URL.Query())
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		ids := []string{}
		err = json.NewDecoder(r.Body).Decode(&ids)
		if err != nil {
			log.Println("WARNING: error in user send data", err)
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		result, err := q.GetListFromIds(token, kind, ids, queryListCommons)
		if err != nil {
			log.Println("ERROR:", ids, err)
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(result)
	})

	router.POST("/ids/select/:resource_kind/:right/:limit/:offset/:orderfeature/:direction", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		limit := ps.ByName("limit")
		offset := ps.ByName("offset")
		orderfeature := ps.ByName("orderfeature")
		direction := ps.ByName("direction")
		ids := []string{}
		err := json.NewDecoder(r.Body).Decode(&ids)
		if err != nil {
			log.Println("WARNING: error in user send data", err)
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		queryListCommons, err := model.GetQueryListCommonsFromStrings(limit, offset, orderfeature, direction, right)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		result, err := q.GetListFromIds(token, kind, ids, queryListCommons)
		if err != nil {
			log.Println("ERROR:", ids, err)
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(result)
	})

	router.GET("/user/list/:user/:resource_kind/:right", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		user := ps.ByName("user")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		list, err := q.GetListForUser(kind, user, right)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(list)
	})

	router.GET("/user/check/:user/:resource_kind/:resource_id/:right", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		user := ps.ByName("user")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		resource := ps.ByName("resource_id")
		err := q.CheckUser(kind, resource, user, right)
		if err != nil {
			log.Println("access denied", err)
			http.Error(res, "access denied", http.StatusUnauthorized)
			return
		}
		ok := map[string]string{"status": "ok"}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(ok)
	})

	router.GET("/group/list/:group/:resource_kind/:right", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		group := ps.ByName("group")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		list, err := q.GetListForGroup(kind, []string{group}, right)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(list)
	})

	router.GET("/group/check/:group/:resource_kind/:resource_id/:right", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		group := ps.ByName("group")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		resource := ps.ByName("resource_id")
		err := q.CheckGroups(kind, resource, []string{group}, right)
		if err != nil {
			log.Println("access denied", err)
			http.Error(res, "access denied", http.StatusUnauthorized)
			return
		}
		ok := map[string]string{"status": "ok"}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(ok)
	})

	router.GET("/export", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		exports, err := q.Export()
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(exports)
	})

	router.PUT("/import", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		imports := map[string][]model.ResourceRights{}
		err := json.NewDecoder(r.Body).Decode(&imports)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		err = q.Import(imports)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		ok := map[string]string{"status": "ok"}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(ok)
	})

	router.POST("/jwt/search/:resource_kind/:query/:right/:limit/:offset/:orderfeature/asc", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		query := ps.ByName("query")
		limit := ps.ByName("limit")
		offset := ps.ByName("offset")
		order := ps.ByName("orderfeature")
		selection := model.Selection{}
		err := json.NewDecoder(r.Body).Decode(&selection)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		queryListCommons, err := model.GetQueryListCommonsFromStrings(limit, offset, order, "asc", right)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		list, err := q.SearchList(token, kind, query, queryListCommons, &selection)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(list)
	})

	router.POST("/jwt/search/:resource_kind/:query/:right/:limit/:offset/:orderfeature/desc", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		query := ps.ByName("query")
		limit := ps.ByName("limit")
		offset := ps.ByName("offset")
		order := ps.ByName("orderfeature")
		selection := model.Selection{}
		err := json.NewDecoder(r.Body).Decode(&selection)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		queryListCommons, err := model.GetQueryListCommonsFromStrings(limit, offset, order, "desc", right)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		list, err := q.SearchList(token, kind, query, queryListCommons, &selection)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(list)
	})

	router.POST("/jwt/list/:resource_kind/:right/:limit/:offset/:orderfeature/asc", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		limit := ps.ByName("limit")
		offset := ps.ByName("offset")
		orderfeature := ps.ByName("orderfeature")
		selection := model.Selection{}
		err := json.NewDecoder(r.Body).Decode(&selection)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		queryListCommons, err := model.GetQueryListCommonsFromStrings(limit, offset, orderfeature, "asc", right)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		list, err := q.GetListWithSelection(token, kind, queryListCommons, selection)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(list)
	})

	router.POST("/jwt/list/:resource_kind/:right/:limit/:offset/:orderfeature/desc", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Deprecation", "true")
		kind := ps.ByName("resource_kind")
		right := ps.ByName("right")
		limit := ps.ByName("limit")
		offset := ps.ByName("offset")
		orderfeature := ps.ByName("orderfeature")
		selection := model.Selection{}
		err := json.NewDecoder(r.Body).Decode(&selection)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		queryListCommons, err := model.GetQueryListCommonsFromStrings(limit, offset, orderfeature, "desc", right)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		list, err := q.GetListWithSelection(token, kind, queryListCommons, selection)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(list)
	})

	return true
}
