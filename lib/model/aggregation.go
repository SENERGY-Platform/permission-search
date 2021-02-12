package model

type TermAggregationResultElement struct {
	Term  interface{} `json:"term"`
	Count int64       `json:"count"`
}
