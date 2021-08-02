package lib

import (
	"context"
	"fmt"
	"github.com/olivere/elastic/v7"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestReceiveDevice(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, q, w, err := getTestEnv(ctx, wg)
	if err != nil {
		fmt.Println(err)
		return
	}

	resource := "devices"
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

	deviceMsg, deviceCmd, err := getDeviceTestObj("device1", map[string]interface{}{
		"name": "device_name_1",
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, deviceMsg, deviceCmd)
	if err != nil {
		t.Error(err)
		return
	}

	attr2 := []interface{}{
		map[string]interface{}{"key": "manufacturer", "value": "42"},
		map[string]interface{}{"key": "api_key", "value": "nope"},
	}
	deviceMsg, deviceCmd, err = getDeviceTestObj("device2", map[string]interface{}{
		"name":       "device_name_2",
		"attributes": attr2,
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = w.UpdateFeatures(resource, deviceMsg, deviceCmd)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	e, _, err := q.GetResourceEntry(resource, "device1")
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(e.Features["name"], "device_name_1") {
		t.Error(e)
		return
	}
	if !reflect.DeepEqual(e.Features["attributes"], nil) {
		t.Error(e)
		return
	}

	result, err := q.GetOrderedListForUserOrGroup(resource, "testOwner", []string{}, "r", "3", "0", "name", true)
	if err != nil {
		t.Error(err)
		return
	}
	if len(result) != 2 {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[0]["name"], "device_name_1") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[1]["name"], "device_name_2") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[1]["attributes"], attr2) {
		t.Error(e)
		return
	}
	t.Log(result)
}
