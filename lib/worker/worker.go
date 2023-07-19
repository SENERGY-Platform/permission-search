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

package worker

import (
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/worker/kafka"
	"github.com/opensearch-project/opensearch-go"
	"github.com/opensearch-project/opensearch-go/opensearchutil"
	"log"
	"time"
)

type Worker struct {
	config  configuration.Config
	query   Query
	timeout time.Duration
	bulk    opensearchutil.BulkIndexer
	done    *kafka.Producer
}

func New(ctx context.Context, config configuration.Config, query Query) (result *Worker, err error) {
	var p *kafka.Producer
	if config.KafkaUrl != "" && config.KafkaUrl != "-" {
		p, err = kafka.NewProducer(ctx, config.KafkaUrl, config.DoneTopic, config.Debug)
		if err != nil {
			return nil, err
		}
	}
	timeout, err := time.ParseDuration(config.Timeout)
	if err != nil {
		return nil, err
	}

	bulkFlushInterval, err := time.ParseDuration(config.BulkFlushInterval)
	if err != nil {
		return nil, err
	}

	client := query.GetClient()
	bulk, err := opensearchutil.NewBulkIndexer(opensearchutil.BulkIndexerConfig{
		Client:        client,                      // The OpenSearch client
		NumWorkers:    int(config.BulkWorkerCount), // The number of worker goroutines (default: number of CPUs)
		FlushBytes:    5e+6,                        // The flush threshold in bytes (default: 5M)
		FlushInterval: bulkFlushInterval,
		OnError: func(ctx context.Context, err error) {
			log.Println("ERROR: bulk:", err)
		},
	})
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		bulk.Close(context.Background())
	}()
	return &Worker{
		config:  config,
		query:   query,
		timeout: timeout,
		bulk:    bulk,
		done:    p,
	}, err
}

func (this *Worker) GetClient() *opensearch.Client {
	return this.query.GetClient()
}

func (this *Worker) GetQuery() Query {
	return this.query
}

func (this *Worker) getTimeout() (ctx context.Context) {
	ctx, _ = context.WithTimeout(context.Background(), this.timeout)
	return ctx
}

func (this *Worker) SendDone(msg model.Done) error {
	if this.done == nil {
		return nil
	}
	msg.Handler = "github.com/SENERGY-Platform/permission-search"
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return this.done.Produce(msg.ResourceId, payload)
}
