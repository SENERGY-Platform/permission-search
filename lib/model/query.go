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

package model

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
)

type QueryMessage struct {
	Resource           string         `json:"resource"`
	Find               *QueryFind     `json:"find"`
	ListIds            *QueryListIds  `json:"list_ids"`
	CheckIds           *QueryCheckIds `json:"check_ids"`
	TermAggregate      *string        `json:"term_aggregate"`
	TermAggregateLimit int            `json:"term_aggregate_limit"`
}
type QueryFind struct {
	QueryListCommons
	Search string     `json:"search"`
	Filter *Selection `json:"filter"`
}

type QueryListIds struct {
	QueryListCommons
	Ids []string `json:"ids"`
}

type QueryCheckIds struct {
	Ids    []string `json:"ids"`
	Rights string   `json:"rights"`
}

type QueryListCommons struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`

	// After may only be set if offset is 0
	// this field must be used if offset + limit would exceed 10000
	// when set the results begin after the referenced id and sort value
	// the sort value is used to efficiently locate the start point
	After *ListAfter `json:"after"`

	Rights   string `json:"rights"`
	SortBy   string `json:"sort_by"`
	SortDesc bool   `json:"sort_desc"`

	// AddIdModifier is used to modify the resource id in the result
	// possible modifiers can be found and configured in the configuration.json under result_modifiers
	// example value: url.Values{"service_group_selection": {"a8ee3b1c-4cda-4f0d-9f55-4ef4882ce0af"}}
	AddIdModifier url.Values `json:"add_id_modifier,omitempty"`
}

type ListAfter struct {
	SortFieldValue interface{} `json:"sort_field_value"`
	Id             string      `json:"id"`
}

func (this QueryListCommons) Validate() error {
	if this.Offset < 0 {
		return fmt.Errorf("%w: offset should be at least 0", ErrBadRequest)
	}
	if this.Limit < 0 {
		return fmt.Errorf("%w: imit should be at least 0", ErrBadRequest)
	}
	if this.Limit+this.Offset > 10000 {
		return fmt.Errorf("%w: limit + offset may not be bigger than 10000. pleas use after.id and after.sort_field_value", ErrBadRequest)
	}
	if this.After != nil {
		if this.Offset != 0 {
			return fmt.Errorf("%w: 'offset' should be 0 if 'after' is used", ErrBadRequest)
		}
		if this.After.SortFieldValue == nil {
			return fmt.Errorf("%w: 'after' needs sort_field_value", ErrBadRequest)
		}
		if this.After.Id == "" {
			return fmt.Errorf("%w: 'after' needs id value", ErrBadRequest)
		}
		if this.SortBy == "" {
			return fmt.Errorf("%w: 'after' needs set sort_by value", ErrBadRequest)
		}
	}
	return nil
}

func (this QueryListCommons) QueryValues() (result url.Values) {
	result = map[string][]string{}
	if this.Limit > 0 {
		result["limit"] = []string{strconv.Itoa(this.Limit)}
	}
	if this.Offset > 0 {
		result["offset"] = []string{strconv.Itoa(this.Offset)}
	}
	if this.SortBy != "" {
		sort := this.SortBy
		if this.SortDesc {
			sort = sort + ".desc"
		}
		result["sort"] = []string{sort}
	}
	if this.Rights != "" {
		result["rights"] = []string{this.Rights}
	}
	if this.After != nil {
		if this.After.SortFieldValue != nil {
			temp, err := json.Marshal(this.After.SortFieldValue)
			if err != nil {
				log.Println("WARNING: QueryListCommons.QueryValues() fall back to fmt.Sprint() for this.After.SortFieldValue")
				result["after.sort_field_value"] = []string{fmt.Sprint(this.After.SortFieldValue)}
			} else {
				result["after.sort_field_value"] = []string{string(temp)}
			}
		}
		if this.After.Id != "" {
			result["after.id"] = []string{this.After.Id}
		}
	}
	if len(this.AddIdModifier) > 0 {
		result["add_id_modifier"] = []string{this.AddIdModifier.Encode()}
	}
	return result
}

func GetQueryListCommonsFromUrlQuery(queryParams url.Values) (result QueryListCommons, err error) {
	limit := queryParams.Get("limit")
	if limit == "" {
		limit = "100"
	}
	offset := queryParams.Get("offset")
	if offset == "" {
		offset = "0"
	}
	sort := queryParams.Get("sort")

	right := queryParams.Get("rights")
	if right == "" {
		right = "r"
	}

	after := ListAfter{
		Id: queryParams.Get("after.id"),
	}
	afterSortStr := queryParams.Get("after.sort_field_value")
	if afterSortStr != "" {
		err = json.Unmarshal([]byte(afterSortStr), &after.SortFieldValue)
		if err != nil {
			return
		}
	}
	if after.SortFieldValue != nil || after.Id != "" {
		result.After = &after
	}

	result.Rights = right
	result.SortBy = strings.Split(sort, ".")[0]
	result.SortDesc = strings.HasSuffix(sort, ".desc")

	result.Limit, err = strconv.Atoi(limit)
	if err != nil {
		return
	}
	result.Offset, err = strconv.Atoi(offset)
	if err != nil {
		return
	}

	addIdModifier := queryParams.Get("add_id_modifier")
	if addIdModifier != "" {
		result.AddIdModifier, err = url.ParseQuery(addIdModifier)
		if err != nil {
			return
		}
	}

	err = result.Validate()
	return
}

func GetQueryListCommonsFromStrings(limit string, offset string, sortBy string, sortDir string, rights string) (result QueryListCommons, err error) {
	if limit == "" {
		limit = "100"
	}
	if offset == "" {
		offset = "0"
	}

	result.Rights = rights
	result.SortBy = sortBy
	result.SortDesc = sortDir != "asc"

	result.Limit, err = strconv.Atoi(limit)
	if err != nil {
		return
	}
	result.Offset, err = strconv.Atoi(offset)
	if err != nil {
		return
	}

	err = result.Validate()
	return
}
