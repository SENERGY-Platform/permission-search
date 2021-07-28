package api

import (
	"github.com/SENERGY-Platform/permission-search/lib/auth"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/olivere/elastic/v7"
)

type Query interface {
	//query
	ResourceExists(kind string, resource string) (exists bool, err error)
	GetRightsToAdministrate(kind string, user string, groups []string) (result []model.ResourceRights, err error)
	CheckUserOrGroup(kind string, resource string, user string, groups []string, rights string) (err error)
	CheckListUserOrGroup(kind string, ids []string, user string, groups []string, rights string) (allowed map[string]bool, err error)
	GetListFromIds(kind string, ids []string, user string, groups []string, rights string) (result []map[string]interface{}, err error)
	GetListFromIdsOrdered(kind string, ids []string, user string, groups []string, rights string, limitStr string, offsetStr string, orderfeature string, asc bool) (result []map[string]interface{}, err error)
	GetFullListForUserOrGroup(kind string, user string, groups []string, rights string) (result []map[string]interface{}, err error)
	GetListForUserOrGroup(kind string, user string, groups []string, rights string, limitStr string, offsetStr string) (result []map[string]interface{}, err error)
	GetOrderedListForUserOrGroup(kind string, user string, groups []string, rights string, limitStr string, offsetStr string, orderfeature string, asc bool) (result []map[string]interface{}, err error)
	GetListForUser(kind string, user string, rights string) (result []string, err error)
	CheckUser(kind string, resource string, user string, rights string) (err error)
	GetListForGroup(kind string, groups []string, rights string) (result []string, err error)
	CheckGroups(kind string, resource string, groups []string, rights string) (err error)
	GetResource(kind string, resource string) (result []model.ResourceRights, err error)
	SearchRightsToAdministrate(kind string, user string, groups []string, query string, limitStr string, offsetStr string) (result []model.ResourceRights, err error)
	SearchListAll(kind string, query string, user string, groups []string, rights string) (result []map[string]interface{}, err error)
	SelectByFieldOrdered(kind string, field string, value string, user string, groups []string, rights string, limitStr string, offsetStr string, orderfeature string, asc bool) (result []map[string]interface{}, err error)
	SelectByFieldAll(kind string, field string, value string, user string, groups []string, rights string) (result []map[string]interface{}, err error)
	SearchList(kind string, query string, user string, groups []string, rights string, limitStr string, offsetStr string) (result []map[string]interface{}, err error)
	SearchOrderedList(kind string, query string, user string, groups []string, rights string, orderFeature string, asc bool, limitStr string, offsetStr string) (result []map[string]interface{}, err error)
	SearchOrderedListWithSelection(kind string, query string, user string, groups []string, rights string, orderFeature string, asc bool, limitStr string, offsetStr string, selection elastic.Query) (result []map[string]interface{}, err error)
	GetOrderedListForUserOrGroupWithSelection(kind string, user string, groups []string, rights string, limitStr string, offsetStr string, orderfeature string, asc bool, selection elastic.Query) (result []map[string]interface{}, err error)

	//selection
	GetFilter(token auth.Token, selection model.Selection) (result elastic.Query, err error)

	//migration
	Import(imports map[string][]model.ResourceRights) (err error)
	Export() (exports map[string][]model.ResourceRights, err error)

	GetTermAggregation(kind string, user string, groups []string, rights string, field string) (result []model.TermAggregationResultElement, err error)
}
