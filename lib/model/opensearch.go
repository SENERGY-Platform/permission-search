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

type SearchResult[T any] struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Shards   struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Skipped    int `json:"skipped"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
	Hits Hits[T] `json:"hits"`
}

// AggregationResult should be used if only one kind of aggregation is requested
// example: model.AggregationResult[Entry, TermsAggrT]{}
type AggregationResult[HitT any, AggrT any] struct {
	SearchResult[HitT]
	Aggregations map[string]AggrT `json:"aggregations"`
}

// AggregationsResult should be used if multiple kinds of aggregations are requested
// example: model.AggregationsResult[Entry, struct{Foo TermsAggrT `json:"foo"`; Bar SometingElse `json:"bar"`}]{}
type AggregationsResult[HitT any, AggrT any] struct {
	SearchResult[HitT]
	Aggregations AggrT `json:"aggregations"`
}

type TermsAggrT struct {
	DocCountErrorUpperBound int64 `json:"doc_count_error_upper_bound"`
	SumOtherDocCount        int64 `json:"sum_other_doc_count"`
	Buckets                 []struct {
		Key      string `json:"key"`
		DocCount int64  `json:"doc_count"`
	} `json:"buckets"`
}

type Hits[T any] struct {
	Total    Total       `json:"total"`
	MaxScore interface{} `json:"max_score"`
	Hits     []Hit[T]    `json:"hits"`
}

type Total struct {
	Value    int64  `json:"value"`
	Relation string `json:"relation"`
}

type Hit[T any] struct {
	Index  string        `json:"_index"`
	Id     string        `json:"_id"`
	Score  interface{}   `json:"_score"`
	Source T             `json:"_source"`
	Sort   []interface{} `json:"sort"`
}

type AliasMapping = map[AliasName]AliasWrapper

type AliasName = string

type AliasWrapper struct {
	Aliases map[IndexName]interface{} `json:"aliases"`
}

type IndexName = string

type ReindexRequest struct {
	Source ReindexIndexRef `json:"source"`
	Dest   ReindexIndexRef `json:"dest"`
}

type ReindexIndexRef struct {
	Index string `json:"index"`
}

type ReindexResult struct {
	Took             int  `json:"took"`
	TimedOut         bool `json:"timed_out"`
	Total            int  `json:"total"`
	Updated          int  `json:"updated"`
	Created          int  `json:"created"`
	Deleted          int  `json:"deleted"`
	Batches          int  `json:"batches"`
	VersionConflicts int  `json:"version_conflicts"`
	Noops            int  `json:"noops"`
	Retries          struct {
		Bulk   int `json:"bulk"`
		Search int `json:"search"`
	} `json:"retries"`
	ThrottledMillis      int           `json:"throttled_millis"`
	RequestsPerSecond    float64       `json:"requests_per_second"`
	ThrottledUntilMillis int           `json:"throttled_until_millis"`
	Failures             []interface{} `json:"failures"`
}

type UpdateAliasRequest struct {
	Actions []UpdateAliasAction `json:"actions"`
}

type UpdateAliasAction struct {
	Remove *UpdateAliasIndexMapping `json:"remove,omitempty"`
	Add    *UpdateAliasIndexMapping `json:"add,omitempty"`
}

type UpdateAliasIndexMapping struct {
	Index string `json:"index"`
	Alias string `json:"alias"`
}

type OpenSearchGetResult struct {
	Index       string      `json:"_index"`
	Id          string      `json:"_id"`
	Version     int64       `json:"_version"`
	SeqNo       int64       `json:"_seq_no"`
	PrimaryTerm int64       `json:"_primary_term"`
	Found       bool        `json:"found"`
	Source      interface{} `json:"_source"`
}
