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

package replay

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/opensearchclient"
	"github.com/SENERGY-Platform/permission-search/lib/rigthsproducer"
	"github.com/opensearch-project/opensearch-go"
	"github.com/opensearch-project/opensearch-go/opensearchutil"
	"runtime/debug"
)

var DefaultBatchSize = 1000

func ReplayPermissions(config configuration.Config, args []string) {
	dryrun := false
	if len(args) == 0 || args[0] != "do" {
		fmt.Println("Dry-Run; to execute use 'do' as the first argument (./permission-search replay-permissions do)")
		dryrun = true
	} else {
		args = args[1:]
	}
	topics := config.ResourceList
	if len(args) > 0 {
		topics = args
	}
	client, err := opensearchclient.New(config)
	if err != nil {
		fmt.Println("ERROR:", err)
		debug.PrintStack()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var producer *rigthsproducer.Producer
	if !dryrun {
		producer, err = rigthsproducer.New(ctx, config)
	}
	if err != nil {
		fmt.Println("ERROR:", err)
		debug.PrintStack()
		return
	}
	for _, topic := range topics {
		ReplayPermissionsOfResourceKind(producer, client, topic, DefaultBatchSize)
	}
}

func ReplayPermissionsOfResourceKind(producer *rigthsproducer.Producer, client *opensearch.Client, kind string, batchSize int) {
	for entry := range GetEntries(client, kind, batchSize) {
		rights := entry.ToResourceRights().ResourceRightsBase
		fmt.Printf("%#v %#v %#v\n", kind, entry.Resource, rights)
		if producer != nil {
			err, _ := producer.SetResourceRights(kind, entry.Resource, rights, "")
			if err != nil {
				fmt.Println("ERROR:", err)
				debug.PrintStack()
				return
			}
		}
	}
}

func GetEntries(client *opensearch.Client, kind string, batchSize int) (entries chan model.Entry) {
	lastId := ""
	entries = make(chan model.Entry, batchSize)
	go func() {
		defer close(entries)
		for {
			query := map[string]interface{}{
				"query": map[string]interface{}{
					"match_all": map[string]interface{}{},
				},
			}
			if lastId != "" {
				query["search_after"] = []interface{}{lastId}
			}
			resp, err := client.Search(
				client.Search.WithIndex(kind),
				client.Search.WithVersion(true),
				client.Search.WithSize(batchSize),
				client.Search.WithSort("resource:asc"),
				client.Search.WithBody(opensearchutil.NewJSONReader(query)),
			)
			if err != nil {
				fmt.Println("ERROR:", err)
				debug.PrintStack()
				return
			}
			defer resp.Body.Close()
			if resp.IsError() {
				fmt.Println("ERROR:", resp.String())
				debug.PrintStack()
				return
			}
			pl := model.SearchResult[model.Entry]{}
			err = json.NewDecoder(resp.Body).Decode(&pl)
			if err != nil {
				fmt.Println("ERROR:", err)
				debug.PrintStack()
				return
			}

			for _, hit := range pl.Hits.Hits {
				entry := hit.Source
				entries <- entry
				lastId = entry.Resource
			}
			if len(pl.Hits.Hits) < batchSize {
				return
			}
		}
	}()
	return entries
}
