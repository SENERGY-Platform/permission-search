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

func TestReceiveCharacteristic(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, q, w, err := getTestEnv(ctx, wg)
	if err != nil {
		fmt.Println(err)
		return
	}

	resource := "characteristics"
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
	characteristicMsg, characteristicCmd, err := getcharacteristicTestObj("characteristic1", map[string]interface{}{
		"name":         "characteristic_name",
		"display_unit": "display",
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, characteristicMsg, characteristicCmd)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	e, _, err := q.GetResourceEntry(resource, "characteristic1")
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(e.Features["name"], "characteristic_name") {
		t.Error(e)
		return
	}
	if !reflect.DeepEqual(e.Features["display_unit"], "display") {
		t.Error(e)
		return
	}

	result, err := q.GetOrderedListForUserOrGroup(resource, "testOwner", []string{"user"}, model.QueryListCommons{
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
	if !reflect.DeepEqual(result[0]["name"], "characteristic_name") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[0]["display_unit"], "display") {
		t.Error(result)
		return
	}
}

func getcharacteristicTestObj(id string, obj map[string]interface{}) (msg []byte, command model.CommandWrapper, err error) {
	text := `{
		"command": "PUT",
		"id": "%s",
		"owner": "testOwner",
		"characteristic": %s
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
