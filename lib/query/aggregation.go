package query

import (
	"context"
	"errors"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/olivere/elastic/v7"
)

func (this *Query) GetTermAggregation(kind string, user string, groups []string, rights string, field string, limit int) (result []model.TermAggregationResultElement, err error) {
	ctx := context.Background()
	query := elastic.NewBoolQuery().Filter(getRightsQuery(rights, user, groups)...)
	aggregate := elastic.NewTermsAggregation().Field(field)
	resp, err := this.client.Search().Index(kind).Version(true).Query(query).Aggregation(field, aggregate).Size(limit).Do(ctx)
	if err != nil {
		return result, err
	}
	termsAggregation, found := resp.Aggregations.Terms(field)
	if !found {
		return nil, errors.New("aggregation result not found in response from elasticsearch")
	}
	for _, bucket := range termsAggregation.Buckets {
		result = append(result, model.TermAggregationResultElement{
			Term:  bucket.Key,
			Count: bucket.DocCount,
		})
	}
	return result, err
}
