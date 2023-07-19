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
	"github.com/SENERGY-Platform/permission-search/lib/worker/kafka"
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
	var producer *kafka.Producer
	if !dryrun {
		producer, err = kafka.NewProducer(ctx, config.KafkaUrl, config.PermTopic, true)
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

func ReplayPermissionsOfResourceKind(producer *kafka.Producer, client *opensearch.Client, kind string, batchSize int) {
	for command := range GetCommands(client, kind, batchSize) {
		msg, err := json.Marshal(command)
		if err != nil {
			fmt.Println("ERROR:", err)
			debug.PrintStack()
			return
		}
		fmt.Println(string(msg))
		if producer != nil {
			err = producer.Produce(command.Resource+"_"+command.User+"_"+command.Group, msg)
			if err != nil {
				fmt.Println("ERROR:", err)
				debug.PrintStack()
				return
			}
		}
	}
}

func GetCommands(client *opensearch.Client, kind string, batchSize int) (commands chan model.PermCommandMsg) {
	commands = make(chan model.PermCommandMsg)
	entries := GetEntries(client, kind, batchSize)
	go func() {
		defer close(commands)
		for entry := range entries {
			userRight := map[string]string{}
			for _, user := range entry.ReadUsers {
				userRight[user] = userRight[user] + "r"
			}
			for _, user := range entry.WriteUsers {
				userRight[user] = userRight[user] + "w"
			}
			for _, user := range entry.ExecuteUsers {
				userRight[user] = userRight[user] + "x"
			}
			for _, user := range entry.AdminUsers {
				userRight[user] = userRight[user] + "a"
			}
			for user, right := range userRight {
				commands <- model.PermCommandMsg{
					Command:  "PUT",
					Kind:     kind,
					Resource: entry.Resource,
					User:     user,
					Right:    right,
				}
			}
			groupRight := map[string]string{}
			for _, group := range entry.ReadGroups {
				groupRight[group] = groupRight[group] + "r"
			}
			for _, group := range entry.WriteGroups {
				groupRight[group] = groupRight[group] + "w"
			}
			for _, group := range entry.ExecuteGroups {
				groupRight[group] = groupRight[group] + "x"
			}
			for _, group := range entry.AdminGroups {
				groupRight[group] = groupRight[group] + "a"
			}
			for group, right := range groupRight {
				commands <- model.PermCommandMsg{
					Command:  "PUT",
					Kind:     kind,
					Resource: entry.Resource,
					Group:    group,
					Right:    right,
				}
			}

		}
	}()
	return commands
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
