/*
 * Copyright 2023 InfAI (CC SES)
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

package client

import (
	"bytes"
	"encoding/json"
	"github.com/SENERGY-Platform/permission-search/lib/auth"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"net/http"
)

func (this *impl) GetRights(token auth.Token, kind string, resource string) (result model.ResourceRights, err error) {
	req, err := http.NewRequest(http.MethodGet, this.baseUrl+"/v3/administrate/rights/"+kind+"/"+resource, nil)
	req.Header.Set("Authorization", token.Jwt())
	if err != nil {
		return result, err
	}
	result, _, err = do[model.ResourceRights](req)
	return
}

func (this *impl) Query(token auth.Token, query model.QueryMessage) (result interface{}, code int, err error) {
	buf := &bytes.Buffer{}
	err = json.NewEncoder(buf).Encode(query)
	if err != nil {
		return result, http.StatusInternalServerError, err
	}
	req, err := http.NewRequest(http.MethodGet, this.baseUrl+"/v3/query", nil)
	req.Header.Set("Authorization", token.Jwt())
	if err != nil {
		return result, http.StatusInternalServerError, err
	}
	result, code, err = do[interface{}](req)
	return
}

func (this *impl) List(token auth.Token, kind string, options model.ListOptions) (result []map[string]interface{}, err error) {
	//TODO implement me
	panic("implement me")
}

func (this *impl) Total(token auth.Token, kind string, options model.ListOptions) (result int64, err error) {
	//TODO implement me
	panic("implement me")
}

func (this *impl) CheckUserOrGroup(token auth.Token, kind string, resource string, rights string) (err error) {
	//TODO implement me
	panic("implement me")
}

func (this *impl) CheckListUserOrGroup(token auth.Token, kind string, ids []string, rights string) (allowed map[string]bool, err error) {
	//TODO implement me
	panic("implement me")
}

func (this *impl) GetTermAggregation(token auth.Token, kind string, rights string, field string, limit int) (result []model.TermAggregationResultElement, err error) {
	//TODO implement me
	panic("implement me")
}
