package api

import (
	"encoding/json"
	"github.com/SENERGY-Platform/permission-search/lib/auth"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SmartEnergyPlatform/util/http/response"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"strings"
)

func init() {
	endpoints = append(endpoints, V3Endpoints)
}

func V3Endpoints(router *httprouter.Router, config configuration.Config, q Query) {

	router.GET("/v3/administrate/rights/:resource/:id", func(res http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		resource := ps.ByName("resource")
		id := ps.ByName("id")
		token, err := auth.GetParsedToken(r)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		if err := q.CheckUserOrGroup(resource, id, token.GetUserId(), token.GetRoles(), "a"); err != nil {
			log.Println("access denied", err)
			http.Error(res, "access denied", http.StatusUnauthorized)
			return
		}
		list, err := q.GetResource(resource, id)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		if len(list) == 0 {
			http.Error(res, "404", http.StatusNotFound)
			return
		}
		response.To(res).Json(list[0])
	})

	router.GET("/v3/resources/:resource", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		resource := params.ByName("resource")

		search := request.URL.Query().Get("search")
		selection := request.URL.Query().Get("filter")
		ids := request.URL.Query().Get("ids")

		queryListCommons, err := model.GetQueryListCommonsFromUrlQuery(request.URL.Query())
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

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

		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		var result []map[string]interface{}

		switch mode {
		case "search":
			result, err = q.SearchOrderedList(resource, search, token.GetUserId(), token.GetRoles(), queryListCommons)
		case "selection":
			selectionParts := strings.Split(selection, ":")
			if len(selectionParts) < 2 {
				http.Error(writer, "the query parameter 'select' expects a value like 'field_name:field_value'", http.StatusBadRequest)
				return
			}
			field := selectionParts[0]
			value := strings.Join(selectionParts[1:], ":")
			result, err = q.SelectByFieldOrdered(resource, field, value, token.GetUserId(), token.GetRoles(), queryListCommons)
		case "ids":
			// not more than 10 ids should be send
			result, err = q.GetListFromIdsOrdered(resource, strings.Split(ids, ","), token.GetUserId(), token.GetRoles(), queryListCommons)
		default:
			result, err = q.GetOrderedListForUserOrGroup(resource, token.GetUserId(), token.GetRoles(), queryListCommons)
		}

		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(writer).Encode(result)
	})

	router.HEAD("/v3/resources/:resource/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		resource := params.ByName("resource")
		id := params.ByName("id")
		right := request.URL.Query().Get("rights")
		if right == "" {
			right = "r"
		}
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		err = q.CheckUserOrGroup(resource, id, token.GetUserId(), token.GetRoles(), right)
		if err != nil {
			http.Error(writer, "access denied: "+err.Error(), http.StatusUnauthorized)
			return
		}
		writer.WriteHeader(200)
	})

	router.GET("/v3/resources/:resource/:id/access", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		resource := params.ByName("resource")
		id := params.ByName("id")
		right := request.URL.Query().Get("rights")
		if right == "" {
			right = "r"
		}
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		err = q.CheckUserOrGroup(resource, id, token.GetUserId(), token.GetRoles(), right)
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err != nil {
			json.NewEncoder(writer).Encode(false)
		} else {
			json.NewEncoder(writer).Encode(true)
		}
	})

	router.GET("/v3/aggregates/term/:resource/:term", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		resource := params.ByName("resource")
		term := params.ByName("term")
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		result, err := q.GetTermAggregation(resource, token.GetUserId(), token.GetRoles(), "r", term)

		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(writer).Encode(result)
	})

	router.POST("/v3/query", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		query := model.QueryMessage{}
		err = json.NewDecoder(request.Body).Decode(&query)
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
						token.GetUserId(),
						token.GetRoles(),
						query.Find.QueryListCommons)
				} else {
					filter, err := q.GetFilter(token, *query.Find.Filter)
					if err != nil {
						http.Error(writer, err.Error(), http.StatusBadRequest)
						return
					}
					result, err = q.GetOrderedListForUserOrGroupWithSelection(
						query.Resource,
						token.GetUserId(),
						token.GetRoles(),
						query.Find.QueryListCommons,
						filter)
				}
			} else {
				if query.Find.Filter == nil {
					result, err = q.SearchOrderedList(
						query.Resource,
						query.Find.Search,
						token.GetUserId(),
						token.GetRoles(),
						query.Find.QueryListCommons)
				} else {
					filter, err := q.GetFilter(token, *query.Find.Filter)
					if err != nil {
						http.Error(writer, err.Error(), http.StatusBadRequest)
						return
					}
					result, err = q.SearchOrderedListWithSelection(
						query.Resource,
						query.Find.Search,
						token.GetUserId(),
						token.GetRoles(),
						query.Find.QueryListCommons,
						filter)
				}
			}
		}

		if query.CheckIds != nil {
			result, err = q.CheckListUserOrGroup(
				query.Resource,
				query.CheckIds.Ids,
				token.GetUserId(),
				token.GetRoles(),
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
				token.GetUserId(),
				token.GetRoles(),
				query.ListIds.QueryListCommons)
		}

		if query.TermAggregate != nil {
			result, err = q.GetTermAggregation(query.Resource, token.GetUserId(), token.GetRoles(), "r", *query.TermAggregate)
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
