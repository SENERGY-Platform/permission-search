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
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"

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
