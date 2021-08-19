package model

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
)

type QueryMessage struct {
	Resource      string         `json:"resource"`
	Find          *QueryFind     `json:"find"`
	ListIds       *QueryListIds  `json:"list_ids"`
	CheckIds      *QueryCheckIds `json:"check_ids"`
	TermAggregate *string        `json:"term_aggregate"`
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
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
	Rights   string `json:"rights"`
	SortBy   string `json:"sort_by"`
	SortDesc bool   `json:"sort_desc"`
}

func (this QueryListCommons) Validate() error {
	if this.Offset < 0 {
		return errors.New("offset should be at leas 0")
	}
	if this.Limit < 0 {
		return errors.New("limit should be at leas 0")
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
