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
	"errors"
	"github.com/SENERGY-Platform/permission-search/lib/model"
)

type TestClient struct {
	resourceRights map[string]map[string]model.ResourceRights
}

func NewTestClient() *TestClient {
	return &TestClient{resourceRights: map[string]map[string]model.ResourceRights{}}
}

func (this *TestClient) GetRights(_ string, resource string, id string) (result model.ResourceRights, err error) {
	resources, ok := this.resourceRights[resource]
	if !ok {
		return model.ResourceRights{}, errors.New("not found")
	}
	result, ok = resources[id]
	if !ok {
		return model.ResourceRights{}, errors.New("not found")
	}
	return result, nil
}

func (this *TestClient) SetRights(resource string, id string, rights model.ResourceRights) {
	resources, ok := this.resourceRights[resource]
	if !ok {
		resources = map[string]model.ResourceRights{}
	}
	resources[id] = rights
	this.resourceRights[resource] = resources
}

func (this *TestClient) Query(token string, query QueryMessage) (result interface{}, code int, err error) {
	//TODO implement me
	panic("implement me")
}

func (this *TestClient) List(token string, kind string, options ListOptions) (result []map[string]interface{}, err error) {
	//TODO implement me
	panic("implement me")
}

func (this *TestClient) Total(token string, kind string, options ListOptions) (result int64, err error) {
	//TODO implement me
	panic("implement me")
}

func (this *TestClient) CheckUserOrGroup(token string, kind string, resource string, rights string) (err error) {
	//TODO implement me
	panic("implement me")
}

func (this *TestClient) GetTermAggregation(token string, kind string, rights string, field string, limit int) (result []model.TermAggregationResultElement, err error) {
	//TODO implement me
	panic("implement me")
}
