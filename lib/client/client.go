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
	"encoding/json"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"io"
	"net/http"
)

type Client interface {
	Query(token string, query QueryMessage) (result interface{}, code int, err error)
	List(token string, kind string, options ListOptions) (result []map[string]interface{}, err error)
	Total(token string, kind string, options ListOptions) (result int64, err error)

	CheckUserOrGroup(token string, kind string, resource string, rights string) (err error)
	GetRights(token string, kind string, resource string) (result model.ResourceRights, err error)

	GetTermAggregation(token string, kind string, rights string, field string, limit int) (result []model.TermAggregationResultElement, err error)

	ExportKind(token string, kind string, limit int, offset int) (result []model.ResourceRights, err error)
}

type impl struct {
	baseUrl string
}

func NewClient(baseUrl string) Client {
	return &impl{baseUrl: baseUrl}
}

func do[T any](req *http.Request) (result T, code int, err error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, http.StatusInternalServerError, err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		temp, _ := io.ReadAll(resp.Body) //read error response end ensure that resp.Body is read to EOF
		return result, resp.StatusCode, model.GetErrFromCode(resp.StatusCode, string(temp))
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		_, _ = io.ReadAll(resp.Body) //ensure resp.Body is read to EOF
		return result, http.StatusInternalServerError, err
	}
	return result, resp.StatusCode, nil
}

func head(req *http.Request) (code int, err error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		temp, _ := io.ReadAll(resp.Body) //read error response end ensure that resp.Body is read to EOF
		return resp.StatusCode, model.GetErrFromCode(resp.StatusCode, string(temp))
	}
	return resp.StatusCode, nil
}
