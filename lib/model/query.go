package model

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
)

type QueryMessage struct {
	Resource           string         `json:"resource"`
	Find               *QueryFind     `json:"find"`
	ListIds            *QueryListIds  `json:"list_ids"`
	CheckIds           *QueryCheckIds `json:"check_ids"`
	TermAggregate      *string        `json:"term_aggregate"`
	TermAggregateLimit int            `json:"term_aggregate_limit"`
}
type QueryFind struct {
	QueryListCommons
	Search string     `json:"search"`
	Filter *Selection `json:"filter"`
}

type QueryListIds struct {
	QueryListCommons
	Ids []string `json:"ids"`
}

type QueryCheckIds struct {
	Ids    []string `json:"ids"`
	Rights string   `json:"rights"`
}

type QueryListCommons struct {
	Limit    int        `json:"limit"`
	Offset   int        `json:"offset"`
	After    *ListAfter `json:"after"`
	Rights   string     `json:"rights"`
	SortBy   string     `json:"sort_by"`
	SortDesc bool       `json:"sort_desc"`
}

type ListAfter struct {
	SortFieldValue interface{} `json:"sort_field_value"`
	Id             string      `json:"id"`
}

func (this QueryListCommons) Validate() error {
	if this.Offset < 0 {
		return errors.New("offset should be at least 0")
	}
	if this.Limit < 0 {
		return errors.New("limit should be at least 0")
	}
	if this.Limit+this.Offset > 10000 {
		return errors.New("limit + offset may not be bigger than 10000. pleas use after.id and after.sort_field_value")
	}
	if this.After != nil {
		if this.Offset != 0 {
			return errors.New("'offset' should be 0 if 'after' is used")
		}
		if this.After.SortFieldValue == nil {
			return errors.New("'after' needs sort_field_value")
		}
		if this.After.Id == "" {
			return errors.New("'after' needs id value")
		}
	}
	return nil
}

func GetQueryListCommonsFromUrlQuery(queryParams url.Values) (result QueryListCommons, err error) {
	limit := queryParams.Get("limit")
	if limit == "" {
		limit = "100"
	}
	offset := queryParams.Get("offset")
	if offset == "" {
		offset = "0"
	}
	sort := queryParams.Get("sort")
	if sort == "" {
		sort = "name"
	}

	right := queryParams.Get("rights")
	if right == "" {
		right = "r"
	}

	after := ListAfter{
		Id: queryParams.Get("after.id"),
	}
	afterSortStr := queryParams.Get("after.sort_field_value")
	if afterSortStr != "" {
		err = json.Unmarshal([]byte(afterSortStr), &after.SortFieldValue)
		if err != nil {
			return
		}
	}
	if after.SortFieldValue != nil || after.Id != "" {
		result.After = &after
	}

	result.Rights = right
	result.SortBy = strings.Split(sort, ".")[0]
	result.SortDesc = strings.HasSuffix(sort, ".desc")

	result.Limit, err = strconv.Atoi(limit)
	if err != nil {
		return
	}
	result.Offset, err = strconv.Atoi(offset)
	if err != nil {
		return
	}
	err = result.Validate()
	return
}

func GetQueryListCommonsFromStrings(limit string, offset string, sortBy string, sortDir string, rights string) (result QueryListCommons, err error) {
	if limit == "" {
		limit = "100"
	}
	if offset == "" {
		offset = "0"
	}

	result.Rights = rights
	result.SortBy = sortBy
	result.SortDesc = sortDir != "asc"

	result.Limit, err = strconv.Atoi(limit)
	if err != nil {
		return
	}
	result.Offset, err = strconv.Atoi(offset)
	if err != nil {
		return
	}

	err = result.Validate()
	return
}
