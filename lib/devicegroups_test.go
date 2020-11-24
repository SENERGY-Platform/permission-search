package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	jwt_http_router "github.com/SmartEnergyPlatform/jwt-http-router"
	"github.com/olivere/elastic/v7"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestDeviceGroup(t *testing.T) {
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
	_, err = q.GetClient().DeleteByQuery(resource).Query(elastic.NewMatchAllQuery()).Do(context.Background())
	if err != nil {
		t.Error(err)
		return
	}
	_, err = q.GetClient().Flush().Index(resource).Do(context.Background())
	if err != nil {
		t.Error(err)
		return
	}

	msg, cmd, err := getDeviceGroupTestObj("g1", map[string]interface{}{
		"name":                "g1_name",
		"image":               "g1_image",
		"device_ids":          []string{"d1", "d2"},
		"foo":                 "bar",
		"bar":                 42,
		"blocked_interaction": "event",
		"criteria": []interface{}{
			map[string]interface{}{
				"function_id":     "f1",
				"aspect_id":       "a1",
				"device_class_id": "",
			},
			map[string]interface{}{
				"function_id": "f1",
				"aspect_id":   "a2",
			},
			map[string]interface{}{
				"function_id":     "f2",
				"aspect_id":       "",
				"device_class_id": "dc1",
			},
			map[string]interface{}{
				"function_id":     "f3",
				"device_class_id": "dc1",
			},
		},
		"criteria_short": []interface{}{
			"f1_a1_",
			"f1_a2_",
			"f2__dc1",
			"f3__dc1",
		},
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, msg, cmd)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	e, _, err := q.GetResourceEntry(resource, "g1")
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(e.Features["name"], "g1_name") {
		t.Error(e)
		return
	}

	expected := []map[string]interface{}{{
		"id":                  "g1",
		"name":                "g1_name",
		"image":               "g1_image",
		"device_ids":          []interface{}{"d1", "d2"},
		"blocked_interaction": "event",
		"criteria": []interface{}{
			map[string]interface{}{
				"function_id":     "f1",
				"aspect_id":       "a1",
				"device_class_id": "",
			},
			map[string]interface{}{
				"function_id": "f1",
				"aspect_id":   "a2",
			},
			map[string]interface{}{
				"function_id":     "f2",
				"aspect_id":       "",
				"device_class_id": "dc1",
			},
			map[string]interface{}{
				"function_id":     "f3",
				"device_class_id": "dc1",
			},
		},
		"criteria_short": []interface{}{
			"f1_a1_",
			"f1_a2_",
			"f2__dc1",
			"f3__dc1",
		},
		"creator": "testOwner",
		"permissions": map[string]bool{
			"a": true,
			"r": true,
			"w": true,
			"x": true,
		},
		"shared": false,
	}}

	result, err := q.GetOrderedListForUserOrGroup(resource, "testOwner", []string{"user"}, "r", "3", "0", "name", true)
	if err != nil {
		t.Error(err)
		return
	}
	if len(result) != 1 {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result, expected) {
		j, _ := json.Marshal(result)
		t.Error(string(j))
		return
	}

	filter, err := q.GetFilter(jwt_http_router.Jwt{}, model.Selection{
		And: []model.Selection{
			{
				Condition: model.ConditionConfig{
					Feature:   "features.criteria_short",
					Operation: model.QueryEqualOperation,
					Value:     "f1_a1_",
				},
			},
			{
				Condition: model.ConditionConfig{
					Feature:   "features.criteria_short",
					Operation: model.QueryEqualOperation,
					Value:     "f1_a2_",
				},
			},
		},
	})
	if err != nil {
		t.Error(err)
		return
	}
	result, err = q.GetOrderedListForUserOrGroupWithSelection(
		resource,
		"testOwner",
		[]string{"user"},
		"r",
		"3",
		"0",
		"name",
		true,
		filter)

	if err != nil {
		t.Error(err)
		return
	}
	if len(result) != 1 {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result, expected) {
		j, _ := json.Marshal(result)
		t.Error(string(j))
		return
	}
}

func getDeviceGroupTestObj(id string, obj map[string]interface{}) (msg []byte, command model.CommandWrapper, err error) {
	text := `{
		"command": "PUT",
		"id": "%s",
		"owner": "testOwner",
		"device_group": %s
	}`
	dtStr, err := json.Marshal(obj)
	if err != nil {
		return msg, command, err
	}
	msg = []byte(fmt.Sprintf(text, id, string(dtStr)))
	err = json.Unmarshal(msg, &command)
	if err != nil {
		return msg, command, err
	}
	return msg, command, err
}
