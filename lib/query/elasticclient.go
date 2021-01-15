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
	"context"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"strconv"
	"strings"

	"net/http"
	"syscall"
	"time"

	"log"

	"encoding/json"

	elastic "github.com/olivere/elastic/v7"
)

func (this *Query) GetClient() *elastic.Client {
	return this.client
}

func CreateElasticClient(config configuration.Config) (result *elastic.Client, err error) {
	ctx := context.Background()
	result, err = elastic.NewClient(elastic.SetURL(config.ElasticUrl), elastic.SetRetrier(NewRetrier(config)))
	if err != nil {
		return
	}
	for kind := range config.Resources {
		err = CreateIndex(kind, result, ctx, config)
		if err != nil {
			return
		}
	}
	return
}

func CreateIndex(kind string, client *elastic.Client, ctx context.Context, config configuration.Config) (err error) {
	exists, err := client.IndexExists(kind).Do(ctx)
	if err != nil {
		return err
	}
	mapping, err := model.CreateMapping(config, kind)
	if err != nil {
		return err
	}
	mappingJson, _ := json.Marshal(mapping)
	log.Println("expected index setting ", kind, string(mappingJson))
	if !exists {
		log.Println("create new index")
		createIndex, err := client.CreateIndex(kind + "_v1").BodyJson(mapping).Do(ctx)
		if err != nil {
			return err
		}
		if !createIndex.Acknowledged {
			return errors.New("index not acknowledged")
		}
		_, err = client.Alias().Add(kind+"_v1", kind).Do(ctx)
	}
	return
}

func UpdateIndexes(config configuration.Config, resourceNames ...string) error {
	ctx := context.Background()
	client, err := elastic.NewClient(elastic.SetURL(config.ElasticUrl), elastic.SetRetrier(NewRetrier(config)))
	if err != nil {
		return err
	}
	for _, resource := range resourceNames {
		mapping, err := model.CreateMapping(config, resource)
		if err != nil {
			return err
		}
		err = UpdateIndexMapping(client, ctx, resource, mapping)
		if err != nil {
			return err
		}
	}
	return nil
}

func UpdateIndexMapping(client *elastic.Client, ctx context.Context, kind string, mapping map[string]interface{}) error {
	currentVersion, nextVersion, err := GetIndexVersionsOfAlias(client, ctx, kind)
	if err != nil {
		return err
	}

	createIndex, err := client.CreateIndex(nextVersion).BodyJson(mapping).Do(ctx)
	if err != nil {
		return err
	}
	if !createIndex.Acknowledged {
		return errors.New("index not acknowledged")
	}

	log.Println("new index destination for alias", kind, ":", currentVersion, "-->", nextVersion)

	reindexResult, err := client.Reindex().SourceIndex(currentVersion).DestinationIndex(nextVersion).Do(ctx)
	if err != nil {
		return err
	}
	if len(reindexResult.Failures) > 0 {
		temp, _ := json.Marshal(reindexResult.Failures)
		log.Println("ERROR: reindex failures", string(temp))
		return errors.New("reindex failures")
	}

	log.Println("moved", reindexResult.Total, "entries from", currentVersion, "to", nextVersion)

	//update alias
	aliasResult, err := client.Alias().Action(
		elastic.NewAliasRemoveAction(kind).Index(currentVersion),
		elastic.NewAliasAddAction(kind).Index(nextVersion)).
		Do(ctx)
	if err != nil {
		return err
	}
	if !aliasResult.Acknowledged {
		return errors.New("alias update not acknowledged")
	}

	//remove old index
	deleteResult, err := client.DeleteIndex(currentVersion).Do(ctx)
	if err != nil {
		return err
	}
	if !deleteResult.Acknowledged {
		return errors.New("index delete not acknowledged")
	}
	return nil
}

func GetIndexVersionsOfAlias(client *elastic.Client, ctx context.Context, kind string) (current string, next string, err error) {
	result, err := client.Aliases().Alias(kind).Do(ctx)
	if err != nil {
		return current, next, err
	}
	indexes := result.IndicesByAlias(kind)
	if len(indexes) != 1 {
		err = errors.New(fmt.Sprint("unexpected alias result", kind, indexes))
		return current, next, err
	}
	current = indexes[0]
	version, err := strconv.Atoi(strings.ReplaceAll(current, kind+"_v", ""))
	if err != nil {
		return current, next, err
	}
	version = version + 1
	next = kind + "_v" + strconv.Itoa(version)
	return
}

type MyRetrier struct {
	backoff    elastic.Backoff
	maxRetries int64
}

func NewRetrier(config configuration.Config) *MyRetrier {
	return &MyRetrier{
		backoff:    elastic.NewExponentialBackoff(10*time.Millisecond, 8*time.Second),
		maxRetries: config.ElasticRetry,
	}
}

func (r *MyRetrier) Retry(ctx context.Context, retry int, req *http.Request, resp *http.Response, err error) (time.Duration, bool, error) {
	// Fail hard on a specific error
	if err == syscall.ECONNREFUSED {
		return 0, false, errors.New("Elasticsearch or network down")
	}

	// Stop after n retries
	if int64(retry) >= r.maxRetries {
		return 0, false, nil
	}

	wait, stop := r.backoff.Next(retry)
	return wait, stop, nil
}
