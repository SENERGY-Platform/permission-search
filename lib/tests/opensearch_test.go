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

package tests

import (
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/opensearchclient"
	"github.com/SENERGY-Platform/permission-search/lib/tests/docker"
	"github.com/opensearch-project/opensearch-go"
	"github.com/opensearch-project/opensearch-go/opensearchutil"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestOpenSearchStartup(t *testing.T) {
	config, err := configuration.LoadConfig("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	config.OpenSearchInsecureSkipVerify = true
	config.OpenSearchUsername = "admin"
	config.OpenSearchPassword = "admin"

	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, ip, err := docker.OpenSearch(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.OpenSearchUrls = "https://" + ip + ":9200"

	client, err := opensearchclient.New(config)
	if err != nil {
		t.Error(err)
		return
	}

	entry1 := model.Entry{
		Resource: "foo",
		Features: map[string]interface{}{
			"id":   "foo",
			"name": "foo",
		},
		Annotations:   nil,
		AdminUsers:    []string{"admin", "usr1"},
		AdminGroups:   []string{"admin"},
		ReadUsers:     []string{"admin", "usr1"},
		ReadGroups:    []string{"admin", "user"},
		WriteUsers:    []string{"admin", "usr1"},
		WriteGroups:   []string{"admin"},
		ExecuteUsers:  []string{"admin", "usr1"},
		ExecuteGroups: []string{"admin"},
		Creator:       "usr1",
	}

	entry2 := model.Entry{
		Resource: "bar",
		Features: map[string]interface{}{
			"id":   "bar",
			"name": "bar",
		},
		Annotations:   nil,
		AdminUsers:    []string{"admin", "usr2"},
		AdminGroups:   []string{"admin"},
		ReadUsers:     []string{"admin", "usr2"},
		ReadGroups:    []string{"admin", "user"},
		WriteUsers:    []string{"admin", "usr2"},
		WriteGroups:   []string{"admin"},
		ExecuteUsers:  []string{"admin", "usr2"},
		ExecuteGroups: []string{"admin"},
		Creator:       "usr2",
	}

	entry3 := model.Entry{
		Resource: "baz",
		Features: map[string]interface{}{
			"id":   "baz",
			"name": "baz",
		},
		Annotations:   nil,
		AdminUsers:    []string{"test", "usr3"},
		AdminGroups:   []string{"test"},
		ReadUsers:     []string{"test", "usr3"},
		ReadGroups:    []string{"test"},
		WriteUsers:    []string{"test", "usr3"},
		WriteGroups:   []string{"test"},
		ExecuteUsers:  []string{"test", "usr3"},
		ExecuteGroups: []string{"test"},
		Creator:       "usr3",
	}

	resp, err := client.Index(
		"aspects",
		opensearchutil.NewJSONReader(entry1),
		client.Index.WithContext(ctx),
		client.Index.WithDocumentID(entry1.Resource),
		client.Index.WithRefresh("true"),
	)
	if err != nil {
		t.Error(err)
		return
	}
	if resp.IsError() {
		t.Error(resp.String())
		return
	}

	resp, err = client.Index(
		"aspects",
		opensearchutil.NewJSONReader(entry2),
		client.Index.WithContext(ctx),
		client.Index.WithDocumentID(entry2.Resource),
		client.Index.WithRefresh("true"),
	)
	if err != nil {
		t.Error(err)
		return
	}
	if resp.IsError() {
		t.Error(resp.String())
		return
	}

	resp, err = client.Index(
		"aspects",
		opensearchutil.NewJSONReader(entry3),
		client.Index.WithContext(ctx),
		client.Index.WithDocumentID(entry3.Resource),
		client.Index.WithRefresh("true"),
	)
	if err != nil {
		t.Error(err)
		return
	}
	if resp.IsError() {
		t.Error(resp.String())
		return
	}

	t.Run("admin admin r", runOpenSearchSearch(client, "admin", "admin", "r", []model.Entry{entry1, entry2}))
	t.Run("admin admin w", runOpenSearchSearch(client, "admin", "admin", "w", []model.Entry{entry1, entry2}))
	t.Run("admin admin rw", runOpenSearchSearch(client, "admin", "admin", "rw", []model.Entry{entry1, entry2}))
	t.Run("u0 user r", runOpenSearchSearch(client, "u0", "user", "r", []model.Entry{entry1, entry2}))
	t.Run("u0 user w", runOpenSearchSearch(client, "u0", "user", "w", []model.Entry{}))
	t.Run("u0 user rw", runOpenSearchSearch(client, "u0", "user", "rw", []model.Entry{}))
	t.Run("usr1 user w", runOpenSearchSearch(client, "usr1", "user", "w", []model.Entry{entry1}))
	t.Run("usr1 user r", runOpenSearchSearch(client, "usr1", "user", "r", []model.Entry{entry1, entry2}))
	t.Run("u1 user r", runOpenSearchSearch(client, "u1", "user", "r", []model.Entry{entry1, entry2}))
	t.Run("u1 test r", runOpenSearchSearch(client, "u1", "test", "r", []model.Entry{entry3}))
	t.Run("u1 user,test r", runOpenSearchSearch(client, "u1", "user,test", "r", []model.Entry{entry1, entry2, entry3}))
	t.Run("u1 user,test rw", runOpenSearchSearch(client, "u1", "user,test", "rw", []model.Entry{entry3}))
}

func runOpenSearchSearch(client *opensearch.Client, user string, groups string, right string, expected []model.Entry) func(t *testing.T) {
	return func(t *testing.T) {
		kind := "aspects"
		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		query := map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"filter": getRightsQuery(right, user, strings.Split(groups, ",")),
				},
			},
		}
		resp, err := client.Search(client.Search.WithIndex(kind),
			client.Search.WithContext(ctx),
			client.Search.WithVersion(true),
			client.Search.WithBody(opensearchutil.NewJSONReader(query)),
		)
		if err != nil {
			t.Error(err)
			return
		}
		if resp.IsError() {
			t.Error(resp.String())
			return
		}
		pl := model.SearchResult[model.Entry]{}
		err = json.NewDecoder(resp.Body).Decode(&pl)
		if err != nil {
			t.Error(err)
			return
		}
		result := []model.Entry{}
		for _, hit := range pl.Hits.Hits {
			result = append(result, hit.Source)
		}

		sort.Slice(result, func(i, j int) bool {
			return result[i].Resource < result[j].Resource
		})
		sort.Slice(expected, func(i, j int) bool {
			return expected[i].Resource < expected[j].Resource
		})
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("\n%#v\n%#v\n", result, expected)
		}
	}
}

func getRightsQuery(rights string, user string, groups []string) (result []map[string]interface{}) {
	if rights == "" {
		rights = "r"
	}
	for _, right := range rights {
		switch right {
		case 'a':
			or := []map[string]interface{}{}
			if user != "" {
				or = append(or, map[string]interface{}{
					"term": map[string]interface{}{
						"admin_users": user,
					},
				})
			}
			if len(groups) > 0 {
				or = append(or, map[string]interface{}{
					"terms": map[string]interface{}{
						"admin_groups": groups,
					},
				})
			}
			result = append(result, map[string]interface{}{
				"bool": map[string]interface{}{
					"should": or,
				},
			})
		case 'r':
			or := []map[string]interface{}{}
			if user != "" {
				or = append(or, map[string]interface{}{
					"term": map[string]interface{}{
						"read_users": user,
					},
				})
			}
			if len(groups) > 0 {
				or = append(or, map[string]interface{}{
					"terms": map[string]interface{}{
						"read_groups": groups,
					},
				})
			}
			result = append(result, map[string]interface{}{
				"bool": map[string]interface{}{
					"should": or,
				},
			})
		case 'w':
			or := []map[string]interface{}{}
			if user != "" {
				or = append(or, map[string]interface{}{
					"term": map[string]interface{}{
						"write_users": user,
					},
				})
			}
			if len(groups) > 0 {
				or = append(or, map[string]interface{}{
					"terms": map[string]interface{}{
						"write_groups": groups,
					},
				})
			}
			result = append(result, map[string]interface{}{
				"bool": map[string]interface{}{
					"should": or,
				},
			})
		case 'x':
			or := []map[string]interface{}{}
			if user != "" {
				or = append(or, map[string]interface{}{
					"term": map[string]interface{}{
						"execute_users": user,
					},
				})
			}
			if len(groups) > 0 {
				or = append(or, map[string]interface{}{
					"terms": map[string]interface{}{
						"execute_groups": groups,
					},
				})
			}
			result = append(result, map[string]interface{}{
				"bool": map[string]interface{}{
					"should": or,
				},
			})
		}
	}
	return
}
