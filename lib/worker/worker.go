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
