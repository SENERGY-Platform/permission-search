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

func (this *impl) GetList(token auth.Token, kind string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error) {
	//TODO implement me
	panic("implement me")
}

func (this *impl) GetListFromIds(token auth.Token, kind string, ids []string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error) {
	//TODO implement me
	panic("implement me")
}

func (this *impl) GetListWithSelection(token auth.Token, kind string, queryCommons model.QueryListCommons, selection model.Selection) (result []map[string]interface{}, err error) {
	//TODO implement me
	panic("implement me")
}

func (this *impl) SelectByField(token auth.Token, kind string, field string, value string, queryCommons model.QueryListCommons) (result []map[string]interface{}, err error) {
	//TODO implement me
	panic("implement me")
}

func (this *impl) SearchList(token auth.Token, kind string, query string, queryCommons model.QueryListCommons, selection *model.Selection) (result []map[string]interface{}, err error) {
	//TODO implement me
	panic("implement me")
}

func (this *impl) GetTermAggregation(token auth.Token, kind string, rights string, field string, limit int) (result []model.TermAggregationResultElement, err error) {
	//TODO implement me
	panic("implement me")
}

func (this *impl) SearchListTotal(token auth.Token, kind string, search string, right string) (int64, error) {
	//TODO implement me
	panic("implement me")
}

func (this *impl) SelectByFieldTotal(token auth.Token, kind string, field string, value string, right string) (int64, error) {
	//TODO implement me
	panic("implement me")
}

func (this *impl) GetListTotalForUserOrGroup(token auth.Token, kind string, right string) (int64, error) {
	//TODO implement me
	panic("implement me")
}
