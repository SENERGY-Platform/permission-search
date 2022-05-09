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
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/olivere/elastic/v7"
	"time"
)

type Worker struct {
	config  configuration.Config
	query   Query
	timeout time.Duration
}

func New(config configuration.Config, query Query) (result *Worker, err error) {
	timeout, err := time.ParseDuration(config.ElasticTimeout)
	return &Worker{
		config:  config,
		query:   query,
		timeout: timeout,
	}, err
}

func (this *Worker) GetClient() *elastic.Client {
	return this.query.GetClient()
}

func (this *Worker) GetQuery() Query {
	return this.query
}

func (this *Worker) getTimeout() (ctx context.Context) {
	ctx, _ = context.WithTimeout(context.Background(), this.timeout)
	return ctx
}
