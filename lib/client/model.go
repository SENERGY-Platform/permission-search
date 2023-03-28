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

import "github.com/SENERGY-Platform/permission-search/lib/model"

type ListOptions = model.ListOptions

type FeatureSelection = model.FeatureSelection

type QueryListCommons = model.QueryListCommons

type QueryMessage = model.QueryMessage
type QueryFind = model.QueryFind
type QueryListIds = model.QueryListIds
type QueryCheckIds = model.QueryCheckIds
type ListAfter = model.ListAfter

type QueryOperationType = model.QueryOperationType

const (
	QueryEqualOperation             QueryOperationType = model.QueryEqualOperation
	QueryUnequalOperation           QueryOperationType = model.QueryUnequalOperation
	QueryAnyValueInFeatureOperation QueryOperationType = model.QueryAnyValueInFeatureOperation
)

type ConditionConfig = model.ConditionConfig

type Selection = model.Selection

var ErrNotFound = model.ErrNotFound
var ErrAccessDenied = model.ErrAccessDenied
var ErrBadRequest = model.ErrBadRequest
var ErrInvalidAuth = model.ErrInvalidAuth
