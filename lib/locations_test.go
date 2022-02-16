package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/olivere/elastic/v7"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestReceiveLocation(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, q, w, err := getTestEnv(ctx, wg, t)
	if err != nil {
		fmt.Println(err)
		return
	}

	resource := "locations"
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
	locationMsg, locationCmd, err := getLocationTestObj("location1", map[string]interface{}{
		"name":             "location_name",
		"image":            "image1",
		"device_group_ids": []interface{}{"urn:ses:dg1"},
		"device_ids":       []interface{}{"urn:ses:d1"},
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, locationMsg, locationCmd)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	e, _, err := q.GetResourceEntry(resource, "location1")
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(e.Features["name"], "location_name") {
		t.Error(e)
		return
	}

	result, err := q.GetOrderedListForUserOrGroup(resource, "testOwner", []string{}, model.QueryListCommons{
		Limit:    3,
		Offset:   0,
		Rights:   "r",
		SortBy:   "name",
		SortDesc: false,
	})
	if err != nil {
		t.Error(err)
		return
	}
	if len(result) != 1 {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[0]["name"], "location_name") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[0]["image"], "image1") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[0]["device_group_ids"], []interface{}{"urn:ses:dg1"}) {
		temp, _ := json.Marshal(result[0])
		t.Error(result, "\n", string(temp))
		return
	}
	if !reflect.DeepEqual(result[0]["device_ids"], []interface{}{"urn:ses:d1"}) {
		temp, _ := json.Marshal(result[0])
		t.Error(result, "\n", string(temp))
		return
	}
}

func getLocationTestObj(id string, obj map[string]interface{}) (msg []byte, command model.CommandWrapper, err error) {
	text := `{
		"command": "PUT",
		"id": "%s",
		"owner": "testOwner",
		"location": %s
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
