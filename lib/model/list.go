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

package model

import (
	"fmt"
	"net/url"
	"strings"
)

type ListOptions struct {
	QueryListCommons
	ListIds    []string
	TextSearch string
	Selection  *FeatureSelection
}

type FeatureSelection struct {
	Feature string
	Value   string
}

func (this ListOptions) QueryValues() url.Values {
	result := this.QueryListCommons.QueryValues()
	if len(this.ListIds) > 0 {
		result["ids"] = []string{strings.Join(this.ListIds, ",")}
	}
	if len(this.TextSearch) > 0 {
		result["search"] = []string{this.TextSearch}
	}
	if this.Selection != nil {
		result["filter"] = []string{this.Selection.Feature + ":" + this.Selection.Value}
	}
	return result
}

func (this ListOptions) Validate() error {
	_, err := this.Mode()
	if err != nil {
		return err
	}
	return this.QueryListCommons.Validate()
}

func (this ListOptions) Mode() (mode ListOptionsMode, err error) {
	if this.TextSearch != "" {
		mode = ListOptionsModeTextSearch
	}
	if this.Selection != nil {
		if mode != "" {
			return mode, fmt.Errorf("%w: the ListOptions '%v' and 'Selection' may not be combined", ErrBadRequest, mode)
		}
		mode = ListOptionsModeSelection
	}
	if len(this.ListIds) > 0 {
		if mode != "" {
			return mode, fmt.Errorf("%w: the ListOptions '%v' and 'ListIds' may not be combined", ErrBadRequest, mode)
		}
		mode = ListOptionsModeListIds
	}
	if mode == "" {
		mode = ListOptionsModeDefault
	}
	return mode, nil
}

type ListOptionsMode = string

const ListOptionsModeTextSearch = "TextSearch"
const ListOptionsModeSelection = "Selection"
const ListOptionsModeListIds = "ListIds"
const ListOptionsModeDefault = "Default"
