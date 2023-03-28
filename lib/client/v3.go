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
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"net/http"
	"net/url"
	"strconv"
)

func (this *impl) GetRights(token string, kind string, resource string) (result model.ResourceRights, err error) {
	req, err := http.NewRequest(http.MethodGet, this.baseUrl+"/v3/administrate/rights/"+kind+"/"+resource, nil)
	req.Header.Set("Authorization", token)
	if err != nil {
		return result, err
	}
	result, _, err = do[model.ResourceRights](req)
	return
}

func (this *impl) Query(token string, query model.QueryMessage) (result interface{}, code int, err error) {
	buf := &bytes.Buffer{}
	err = json.NewEncoder(buf).Encode(query)
	if err != nil {
		return result, http.StatusInternalServerError, err
	}
	req, err := http.NewRequest(http.MethodPost, this.baseUrl+"/v3/query", buf)
	req.Header.Set("Authorization", token)
	if err != nil {
		return result, http.StatusInternalServerError, err
	}
	result, code, err = do[interface{}](req)
	return
}

func (this *impl) List(token string, kind string, options model.ListOptions) (result []map[string]interface{}, err error) {
	req, err := http.NewRequest(http.MethodGet, this.baseUrl+"/v3/resources/"+url.PathEscape(kind)+"?"+options.QueryValues().Encode(), nil)
	req.Header.Set("Authorization", token)
	if err != nil {
		return result, err
	}
	result, _, err = do[[]map[string]interface{}](req)
	return
}

func (this *impl) Total(token string, kind string, options model.ListOptions) (result int64, err error) {
	req, err := http.NewRequest(http.MethodGet, this.baseUrl+"/v3/total/"+url.PathEscape(kind)+"?"+options.QueryValues().Encode(), nil)
	req.Header.Set("Authorization", token)
	if err != nil {
		return result, err
	}
	result, _, err = do[int64](req)
	return
}

func (this *impl) CheckUserOrGroup(token string, kind string, resource string, rights string) (err error) {
	req, err := http.NewRequest(http.MethodHead, this.baseUrl+"/v3/resources/"+url.PathEscape(kind)+"/"+url.PathEscape(resource)+"?rights="+rights, nil)
	req.Header.Set("Authorization", token)
	if err != nil {
		return err
	}
	_, err = head(req)
	return
}

func (this *impl) GetTermAggregation(token string, kind string, rights string, term string, limit int) (result []model.TermAggregationResultElement, err error) {
	req, err := http.NewRequest(http.MethodHead, this.baseUrl+"/v3/aggregates/term/"+url.PathEscape(kind)+"/"+url.PathEscape(term)+"?limit="+strconv.Itoa(limit)+"&rights="+rights, nil)
	req.Header.Set("Authorization", token)
	if err != nil {
		return result, err
	}
	result, _, err = do[[]model.TermAggregationResultElement](req)
	return
}
