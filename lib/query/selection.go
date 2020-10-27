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

package query

import (
	"errors"
	"github.com/SENERGY-Platform/permission-search/lib/model"

	"reflect"
	"strings"

	"github.com/SmartEnergyPlatform/jwt-http-router"
	elastic "github.com/olivere/elastic/v7"
)

func (this Query) GetFilter(jwt jwt_http_router.Jwt, selection model.Selection) (result elastic.Query, err error) {
	if len(selection.And) > 0 {
		and := []elastic.Query{}
		for _, sub := range selection.And {
			andElement, err := this.GetFilter(jwt, sub)
			if err != nil {
				return result, err
			}
			and = append(and, andElement)
		}
		result = elastic.NewBoolQuery().Filter(and...)
		return
	}
	if len(selection.Or) > 0 {
		or := []elastic.Query{}
		for _, sub := range selection.Or {
			orElement, err := this.GetFilter(jwt, sub)
			if err != nil {
				return result, err
			}
			or = append(or, orElement)
		}
		result = elastic.NewBoolQuery().Should(or...)
		return
	}
	return this.GetConditionFilter(jwt, selection.Condition)
}

func (this Query) GetConditionFilter(jwt jwt_http_router.Jwt, condition model.ConditionConfig) (elastic.Query, error) {
	val := condition.Value
	if val == nil || val == "" {
		switch condition.Ref {
		case "jwt.user":
			val = jwt.UserId
		case "jwt.groups":
			val = jwt.RealmAccess.Roles
		}
	}
	switch condition.Operation {
	case model.QueryEqualOperation:
		if val == nil || val == "" {
			return elastic.NewBoolQuery().MustNot(elastic.NewExistsQuery(condition.Feature)), nil
		} else {
			return elastic.NewTermQuery(condition.Feature, val), nil
		}
	case model.QueryUnequalOperation:
		if val == nil || val == "" {
			return elastic.NewExistsQuery(condition.Feature), nil
		} else {
			return elastic.NewBoolQuery().MustNot(elastic.NewTermQuery(condition.Feature, val)), nil
		}
	case model.QueryAnyValueInFeatureOperation:
		if reflect.TypeOf(val).Kind() == reflect.String {
			val = strings.Split(val.(string), ",")
		}
		arr, err := InterfaceSlice(val)
		if err != nil {
			return nil, err
		}
		return elastic.NewTermsQuery(condition.Feature, arr...), nil
	}
	return nil, errors.New("unknown query opperation type " + string(condition.Operation))
}
