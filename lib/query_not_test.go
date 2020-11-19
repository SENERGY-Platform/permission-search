package lib

import (
	"context"
	"fmt"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/worker"
	jwt_http_router "github.com/SmartEnergyPlatform/jwt-http-router"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestQueryNot(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, q, w, err := getTestEnv(ctx, wg)
	if err != nil {
		fmt.Println(err)
		return
	}

	resource := "device-groups"

	t.Run("create dg1", saveTestDeviceGroup(w, resource, "dg1", map[string]interface{}{"name": "find_dg1_name"}))
	t.Run("create dg2", saveTestDeviceGroup(w, resource, "dg2", map[string]interface{}{"name": "dg2_name"}))
	t.Run("create dg3", saveTestDeviceGroup(w, resource, "dg3", map[string]interface{}{"name": "dg3_name"}))
	t.Run("create dg4", saveTestDeviceGroup(w, resource, "dg4", map[string]interface{}{"name": "find_dg4_name"}))
	t.Run("create dg5", saveTestDeviceGroup(w, resource, "dg5", map[string]interface{}{"name": "find_dg5_name"}))
	t.Run("create dg6", saveTestDeviceGroup(w, resource, "dg6", map[string]interface{}{"name": "dg6_name"}))
	t.Run("create dg7", saveTestDeviceGroup(w, resource, "dg7", map[string]interface{}{"name": "dg7_name"}))

	time.Sleep(2 * time.Second)

	filter, err := q.GetFilter(jwt_http_router.Jwt{}, model.Selection{Not: &model.Selection{Condition: model.ConditionConfig{
		Feature:   "id",
		Operation: model.QueryAnyValueInFeatureOperation,
		Value:     []string{"dg2", "dg4", "dg5"},
	}}})
	if err != nil {
		t.Error(err)
		return
	}

	result, err := q.GetOrderedListForUserOrGroupWithSelection(
		resource,
		"testOwner",
		[]string{"user"},
		"r",
		"100",
		"0",
		"name",
		true,
		filter)
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("checks list", func(t *testing.T) {
		if len(result) != 4 {
			t.Error(len(result), result)
			return
		}
		if !reflect.DeepEqual(result[0]["name"], "dg3_name") {
			t.Error(result)
			return
		}
		if !reflect.DeepEqual(result[1]["name"], "dg6_name") {
			t.Error(result)
			return
		}
		if !reflect.DeepEqual(result[2]["name"], "dg7_name") {
			t.Error(result)
			return
		}
		if !reflect.DeepEqual(result[3]["name"], "find_dg1_name") {
			t.Error(result)
			return
		}
	})

	filter, err = q.GetFilter(jwt_http_router.Jwt{}, model.Selection{Not: &model.Selection{Condition: model.ConditionConfig{
		Feature:   "id",
		Operation: model.QueryAnyValueInFeatureOperation,
		Value:     []string{"dg1"},
	}}})
	if err != nil {
		t.Error(err)
		return
	}

	result, err = q.SearchOrderedListWithSelection(
		resource,
		"find",
		"testOwner",
		[]string{"user"},
		"r",
		"name",
		true,
		"100",
		"0",
		filter)

	if err != nil {
		t.Error(err)
		return
	}

	t.Run("checks search", func(t *testing.T) {
		if len(result) != 2 {
			t.Error(len(result), result)
			return
		}
		if !reflect.DeepEqual(result[0]["name"], "find_dg4_name") {
			t.Error(result)
			return
		}
		if !reflect.DeepEqual(result[1]["name"], "find_dg5_name") {
			t.Error(result)
			return
		}
	})
}

func saveTestDeviceGroup(w *worker.Worker, resource string, id string, fields map[string]interface{}) func(t *testing.T) {
	return func(t *testing.T) {
		msg, cmd, err := getDeviceGroupTestObj(id, fields)
		if err != nil {
			t.Error(err)
			return
		}
		err = w.UpdateFeatures(resource, msg, cmd)
		if err != nil {
			t.Error(err)
			return
		}
	}
}
