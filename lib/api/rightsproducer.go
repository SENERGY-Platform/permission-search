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
)

func init() {
	endpoints = append(endpoints, RightsProducerEndpoints)
}

func RightsProducerEndpoints(router *httprouter.Router, config configuration.Config, q Query, p *rigthsproducer.Producer) bool {
	if p == nil {
		return false
	}
	router.PUT("/v3/administrate/rights/:resource/:id", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		resource := ps.ByName("resource")
		id := ps.ByName("id")
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		key := r.URL.Query().Get("key")
		rights := model.ResourceRightsBase{}
		err = json.NewDecoder(r.Body).Decode(&rights)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		if !token.IsAdmin() {
			if err := q.CheckUserOrGroup(resource, id, token.GetUserId(), token.GetRoles(), "a"); err != nil {
				log.Println("access denied", err)
				http.Error(res, "access denied", http.StatusUnauthorized)
				return
			}
			if !rights.UserRights[token.GetUserId()].Administrate {
				http.Error(res, "user may not remove his own admin rights", http.StatusBadRequest)
				return
			}
		}
		err, code := p.SetResourceRights(resource, id, rights, key)
		if err != nil {
			http.Error(res, err.Error(), code)
			return
		}
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(rights)
	})

	return true
}
