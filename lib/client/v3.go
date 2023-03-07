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
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"net/http"
)

func (this *Client) GetRights(token string, resource string, id string) (result model.ResourceRights, code int, err error) {
	req, err := http.NewRequest(http.MethodGet, this.baseUrl+"/v3/administrate/rights/"+resource+"/"+id, nil)
	req.Header.Set("Authorization", token)
	if err != nil {
		return result, http.StatusInternalServerError, err
	}
	return do[model.ResourceRights](req)
}
