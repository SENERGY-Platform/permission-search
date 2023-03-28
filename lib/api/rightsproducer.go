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
	"errors"
	"github.com/SENERGY-Platform/permission-search/lib/auth"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/query/modifier"
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
		pureId, _ := modifier.SplitModifier(id)
		if pureId != id {
			http.Error(res, "rights con only be changed for ids without '"+modifier.Seperator+"' result-modifier query parts", http.StatusBadRequest)
			return
		}
		if !token.IsAdmin() {
			if err := q.CheckUserOrGroup(token.Jwt(), resource, id, "a"); err != nil {
				log.Println("access denied", err)
				http.Error(res, "access denied", http.StatusForbidden)
				return
			}
			if err = invalidAdminRemoval(rights, token); err != nil {
				http.Error(res, err.Error(), http.StatusBadRequest)
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

func invalidAdminRemoval(rights model.ResourceRightsBase, token auth.Token) error {
	if rights.UserRights[token.GetUserId()].Administrate {
		return nil
	}
	adminByGroup := false
	for group, right := range rights.GroupRights {
		if right.Administrate && token.HasRole(group) {
			adminByGroup = true
			break
		}
	}
	if !adminByGroup {
		return errors.New("user may not remove his own admin ability")
	}

	for _, right := range rights.UserRights {
		if right.Administrate {
			return nil //at least one admin user is kept
		}
	}

	return errors.New("at least one admin user has to be kept")
}
