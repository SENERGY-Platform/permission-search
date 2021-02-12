package api

import (
	"encoding/json"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	jwt_http_router "github.com/SmartEnergyPlatform/jwt-http-router"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func init() {
	endpoints = append(endpoints, V2Endpoints)
}

func V2Endpoints(router *jwt_http_router.Router, config configuration.Config, q Query) {

	router.GET("/v2/:resource", func(writer http.ResponseWriter, request *http.Request, params jwt_http_router.Params, jwt jwt_http_router.Jwt) {
		resource := params.ByName("resource")
		limit := request.URL.Query().Get("limit")
		if limit == "" {
			limit = "100"
		}
		offset := request.URL.Query().Get("offset")
		if offset == "" {
			offset = "0"
		}
		right := request.URL.Query().Get("rights")
		if right == "" {
			right = "r"
		}
		sort := request.URL.Query().Get("sort")
		if sort == "" {
			sort = "name"
		}
		orderby := strings.Split(sort, ".")[0]
		asc := !strings.HasSuffix(sort, ".desc")

		search := request.URL.Query().Get("search")
		selection := request.URL.Query().Get("filter")
		ids := request.URL.Query().Get("ids")

		mode := ""
		if search != "" {
			mode = "search"
		}
		if selection != "" {
			if mode != "" {
				http.Error(writer, "the query parameters "+mode+" and 'select' may not be combined", http.StatusBadRequest)
				return
			}
			mode = "selection"
		}
		if ids != "" {
			if mode != "" {
				http.Error(writer, "the query parameters "+mode+" and 'ids' may not be combined", http.StatusBadRequest)
				return
			}
			mode = "ids"
		}

		var result []map[string]interface{}
		var err error

		switch mode {
		case "search":
			result, err = q.SearchOrderedList(resource, search, jwt.UserId, jwt.RealmAccess.Roles, right, orderby, asc, limit, offset)
		case "selection":
			selectionParts := strings.Split(selection, ":")
			if len(selectionParts) < 2 {
				http.Error(writer, "the query parameter 'select' expects a value like 'field_name:field_value'", http.StatusBadRequest)
				return
			}
			field := selectionParts[0]
			value := strings.Join(selectionParts[1:], ":")
			result, err = q.SelectByFieldOrdered(resource, field, value, jwt.UserId, jwt.RealmAccess.Roles, right, limit, offset, orderby, asc)
		case "ids":
			// not more than 10 ids should be send
			result, err = q.GetListFromIdsOrdered(resource, strings.Split(ids, ","), jwt.UserId, jwt.RealmAccess.Roles, right, limit, offset, orderby, asc)
		default:
			result, err = q.GetOrderedListForUserOrGroup(resource, jwt.UserId, jwt.RealmAccess.Roles, right, limit, offset, orderby, asc)
		}

		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(writer).Encode(result)
	})

	router.HEAD("/v2/:resource/:id", func(writer http.ResponseWriter, request *http.Request, params jwt_http_router.Params, jwt jwt_http_router.Jwt) {
		resource := params.ByName("resource")
		id := params.ByName("id")
		right := request.URL.Query().Get("rights")
		if right == "" {
			right = "r"
		}
		err := q.CheckUserOrGroup(resource, id, jwt.UserId, jwt.RealmAccess.Roles, right)
		if err != nil {
			http.Error(writer, "access denied: "+err.Error(), http.StatusUnauthorized)
			return
		}
		writer.WriteHeader(200)
	})

	router.GET("/v2/:resource/:id/access", func(writer http.ResponseWriter, request *http.Request, params jwt_http_router.Params, jwt jwt_http_router.Jwt) {
		resource := params.ByName("resource")
		id := params.ByName("id")
		right := request.URL.Query().Get("rights")
		if right == "" {
			right = "r"
		}
		err := q.CheckUserOrGroup(resource, id, jwt.UserId, jwt.RealmAccess.Roles, right)
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err != nil {
			json.NewEncoder(writer).Encode(false)
		} else {
			json.NewEncoder(writer).Encode(true)
		}
	})

	router.POST("/v2/query", func(writer http.ResponseWriter, request *http.Request, params jwt_http_router.Params, jwt jwt_http_router.Jwt) {
		query := model.QueryMessage{}
		err := json.NewDecoder(request.Body).Decode(&query)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		if config.Debug {
			temp, _ := json.Marshal(query)
			log.Println("DEBUG:", string(temp))
		}
		var result interface{}
		if query.Find != nil {
			if query.Find.Limit == 0 {
				query.Find.Limit = 100
			}
			if query.Find.SortBy == "" {
				query.Find.SortBy = "name"
			}
			if query.Find.Rights == "" {
				query.Find.Rights = "r"
			}
			if query.Find.Search == "" {
				if query.Find.Filter == nil {
					result, err = q.GetOrderedListForUserOrGroup(
						query.Resource,
						jwt.UserId,
						jwt.RealmAccess.Roles,
						query.Find.Rights,
						strconv.Itoa(query.Find.Limit),
						strconv.Itoa(query.Find.Offset),
						query.Find.SortBy,
						!query.Find.SortDesc)
				} else {
					filter, err := q.GetFilter(jwt, *query.Find.Filter)
					if err != nil {
						http.Error(writer, err.Error(), http.StatusBadRequest)
						return
					}
					result, err = q.GetOrderedListForUserOrGroupWithSelection(
						query.Resource,
						jwt.UserId,
						jwt.RealmAccess.Roles,
						query.Find.Rights,
						strconv.Itoa(query.Find.Limit),
						strconv.Itoa(query.Find.Offset),
						query.Find.SortBy,
						!query.Find.SortDesc,
						filter)
				}
			} else {
				if query.Find.Filter == nil {
					result, err = q.SearchOrderedList(
						query.Resource,
						query.Find.Search,
						jwt.UserId,
						jwt.RealmAccess.Roles,
						query.Find.Rights,
						query.Find.SortBy,
						!query.Find.SortDesc,
						strconv.Itoa(query.Find.Limit),
						strconv.Itoa(query.Find.Offset))
				} else {
					filter, err := q.GetFilter(jwt, *query.Find.Filter)
					if err != nil {
						http.Error(writer, err.Error(), http.StatusBadRequest)
						return
					}
					result, err = q.SearchOrderedListWithSelection(
						query.Resource,
						query.Find.Search,
						jwt.UserId,
						jwt.RealmAccess.Roles,
						query.Find.Rights,
						query.Find.SortBy,
						!query.Find.SortDesc,
						strconv.Itoa(query.Find.Limit),
						strconv.Itoa(query.Find.Offset),
						filter)
				}
			}
		}

		if query.CheckIds != nil {
			result, err = q.CheckListUserOrGroup(
				query.Resource,
				query.CheckIds.Ids,
				jwt.UserId,
				jwt.RealmAccess.Roles,
				query.CheckIds.Rights)
		}

		if query.ListIds != nil {
			if query.ListIds.Limit == 0 {
				query.ListIds.Limit = 100
			}
			if query.ListIds.SortBy == "" {
				query.ListIds.SortBy = "name"
			}
			if query.ListIds.Rights == "" {
				query.ListIds.Rights = "r"
			}
			result, err = q.GetListFromIdsOrdered(
				query.Resource,
				query.ListIds.Ids,
				jwt.UserId,
				jwt.RealmAccess.Roles,
				query.ListIds.Rights,
				strconv.Itoa(query.ListIds.Limit),
				strconv.Itoa(query.ListIds.Offset),
				query.ListIds.SortBy,
				!query.ListIds.SortDesc)
		}

		if query.TermAggregate != nil {
			result, err = q.GetTermAggregation(query.Resource, jwt.UserId, jwt.RealmAccess.Roles, "r", *query.TermAggregate)
		}

		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		if config.Debug {
			temp, _ := json.Marshal(result)
			log.Println("DEBUG:", string(temp))
		}

		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(writer).Encode(result)
	})

}
