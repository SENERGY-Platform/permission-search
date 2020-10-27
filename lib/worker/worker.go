package worker

import (
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/olivere/elastic/v7"
)

type Worker struct {
	config configuration.Config
	query  Query
}

func New(config configuration.Config, query Query) *Worker {
	return &Worker{
		config: config,
		query:  query,
	}
}

func (this *Worker) GetClient() *elastic.Client {
	return this.query.GetClient()
}

func (this *Worker) GetQuery() Query {
	return this.query
}
