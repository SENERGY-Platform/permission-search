package model

type QueryMessage struct {
	Resource string         `json:"resource"`
	Find     *QueryFind     `json:"find"`
	ListIds  *QueryListIds  `json:"list_ids"`
	CheckIds *QueryCheckIds `json:"check_ids"`
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
