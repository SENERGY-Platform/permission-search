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
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/rigthsproducer"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func init() {
	endpoints = append(endpoints, Health)
}

func Health(router *httprouter.Router, config configuration.Config, q Query, p *rigthsproducer.Producer) bool {

	router.GET("/", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		res.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(res).Encode(map[string]interface{}{"ok": true})
	})

	return true
}
