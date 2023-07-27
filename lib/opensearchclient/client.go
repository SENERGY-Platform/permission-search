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

package opensearchclient

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/opensearch-project/opensearch-go"
	"github.com/opensearch-project/opensearch-go/opensearchutil"
	"strconv"
	"strings"

	"net/http"
	"time"

	"log"

	"encoding/json"
)

func New(config configuration.Config) (client *opensearch.Client, err error) {
	ctx := context.Background()
	var discoverNodesInterval time.Duration
	if config.DiscoverOpenSearchNodesInterval != "" {
		discoverNodesInterval, err = time.ParseDuration(config.DiscoverOpenSearchNodesInterval)
	}
	client, err = opensearch.NewClient(opensearch.Config{
		EnableRetryOnTimeout:  true,
		MaxRetries:            config.MaxRetry,
		RetryBackoff:          func(i int) time.Duration { return time.Duration(i) * 100 * time.Millisecond },
		DiscoverNodesOnStart:  config.DiscoverOpenSearchNodesOnStart,
		DiscoverNodesInterval: discoverNodesInterval,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: config.OpenSearchInsecureSkipVerify},
		},
		Addresses: strings.Split(config.OpenSearchUrls, ","),
		Username:  config.OpenSearchUsername, // For testing only. Don't store credentials in code.
		Password:  config.OpenSearchPassword,
	})
	if err != nil {
		log.Println("ERROR: unable to connect to open-search-cluster", err)
		return
	}
	for kind := range config.Resources {
		err = CreateIndex(kind, client, ctx, config)
		if err != nil {
			log.Println("ERROR: unable to create indexes", err)
			return
		}
	}
	return client, nil
}

func CreateIndex(kind string, client *opensearch.Client, ctx context.Context, config configuration.Config) (err error) {
	resp, err := client.Indices.Exists([]string{kind}, client.Indices.Exists.WithContext(ctx))
	if err != nil {
		return err
	}
	mapping, err := model.CreateMapping(config, kind)
	if err != nil {
		return err
	}
	mappingJson, _ := json.Marshal(mapping)
	log.Println("expected index setting ", kind, string(mappingJson))
	if resp.StatusCode == http.StatusNotFound {
		log.Println("create new index")
		resp, err := client.Indices.Create(kind+"_v1", client.Indices.Create.WithBody(opensearchutil.NewJSONReader(mapping)), client.Indices.Create.WithContext(ctx))
		if err != nil {
			return err
		}
		if resp.IsError() {
			return errors.New(resp.String())
		}
		if resp.StatusCode != http.StatusOK {
			return errors.New("index not acknowledged")
		}
		resp, err = client.Indices.PutAlias([]string{kind + "_v1"}, kind, client.Indices.PutAlias.WithContext(ctx))
		if err != nil {
			return err
		}
		if resp.IsError() {
			return errors.New(resp.String())
		}
	}
	return
}

func UpdateIndexes(config configuration.Config, resourceNames ...string) error {
	client, err := New(config)
	if err != nil {
		log.Println("ERROR: unable to connect to database", err)
		return err
	}
	ctx := context.Background()
	for _, resource := range resourceNames {
		mapping, err := model.CreateMapping(config, resource)
		if err != nil {
			return err
		}
		err = UpdateIndexMapping(client, ctx, resource, mapping)
		if err != nil {
			log.Println("ERROR: unable to update index of", resource, err)
			return err
		}
	}
	return nil
}

func UpdateIndexMapping(client *opensearch.Client, ctx context.Context, kind string, mapping map[string]interface{}) error {
	currentVersion, nextVersion, err := GetIndexVersionsOfAlias(client, ctx, kind)
	if err != nil {
		return err
	}

	resp, err := client.Indices.Create(nextVersion, client.Indices.Create.WithBody(opensearchutil.NewJSONReader(mapping)), client.Indices.Create.WithContext(ctx))
	if err != nil {
		return err
	}
	if resp.IsError() {
		return errors.New(resp.String())
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("index not acknowledged")
	}

	log.Println("new index destination for alias", kind, ":", currentVersion, "-->", nextVersion)

	resp, err = client.Reindex(opensearchutil.NewJSONReader(model.ReindexRequest{
		Source: model.ReindexIndexRef{Index: currentVersion},
		Dest:   model.ReindexIndexRef{Index: nextVersion}}),
		client.Reindex.WithContext(ctx),
		client.Reindex.WithRefresh(true))
	if err != nil {
		return err
	}
	if resp.IsError() {
		return errors.New(resp.String())
	}
	reindexResult := model.ReindexResult{}
	err = json.NewDecoder(resp.Body).Decode(&reindexResult)
	if err != nil {
		return err
	}
	if len(reindexResult.Failures) > 0 {
		log.Println("ERROR: reindex failures", resp.String())
		return errors.New("reindex failures")
	}

	log.Println("moved", reindexResult.Total, "entries from", currentVersion, "to", nextVersion)

	//update alias
	resp, err = client.Indices.UpdateAliases(opensearchutil.NewJSONReader(model.UpdateAliasRequest{
		Actions: []model.UpdateAliasAction{
			{
				Remove: &model.UpdateAliasIndexMapping{
					Index: currentVersion,
					Alias: kind,
				},
			},
			{
				Add: &model.UpdateAliasIndexMapping{
					Index: nextVersion,
					Alias: kind,
				},
			},
		},
	}), client.Indices.UpdateAliases.WithContext(ctx))

	if err != nil {
		return err
	}
	if resp.IsError() {
		return errors.New(resp.String())
	}

	//remove old index
	resp, err = client.Indices.Delete([]string{currentVersion}, client.Indices.Delete.WithContext(ctx))
	if err != nil {
		return err
	}
	if resp.IsError() {
		return errors.New(resp.String())
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("index delete not acknowledged")
	}
	return nil

}

func GetIndexVersionsOfAlias(client *opensearch.Client, ctx context.Context, kind string) (current string, next string, err error) {
	resp, err := client.Indices.GetAlias(client.Indices.GetAlias.WithName(kind), client.Indices.GetAlias.WithContext(ctx))
	if err != nil {
		return current, next, err
	}
	if resp.IsError() {
		return current, next, errors.New(resp.String())
	}
	mapping := model.AliasMapping{}
	err = json.NewDecoder(resp.Body).Decode(&mapping)
	if err != nil {
		return current, next, err
	}
	indexes := indicesWithAlias(mapping, kind)

	if len(indexes) != 1 {
		err = errors.New(fmt.Sprintf("unexpected alias result %v %#v \n%#v", kind, indexes, mapping))
		return current, next, err
	}

	current = indexes[0]
	version, err := strconv.Atoi(strings.ReplaceAll(current, kind+"_v", ""))
	if err != nil {
		return current, next, err
	}
	version = version + 1
	next = kind + "_v" + strconv.Itoa(version)

	return current, next, nil
}

func indicesWithAlias(mapping model.AliasMapping, alias string) (result []string) {
	for index, aliases := range mapping {
		for indexAlias, _ := range aliases.Aliases {
			if indexAlias == alias {
				result = append(result, index)
				break
			}
		}
	}
	return result
}
