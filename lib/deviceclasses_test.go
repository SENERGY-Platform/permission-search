package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/olivere/elastic/v7"
	"reflect"
	"testing"
	"time"
)

//dependencies created by TestMain()

func TestReceiveDeviceClass(t *testing.T) {
	resource := "device-classes"
	_, err := GetClient().DeleteByQuery(resource).Query(elastic.NewMatchAllQuery()).Do(context.Background())
	if err != nil {
		t.Error(err)
		return
	}
	_, err = client.Flush().Index(resource).Do(context.Background())
	if err != nil {
		t.Error(err)
		return
	}
	deviceClassMsg, deviceClassCmd, err := getDeviceClassTestObj("deviceClass1", map[string]interface{}{
		"name":  "deviceClass_name",
		"image": "image1",
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = UpdateFeatures(resource, deviceClassMsg, deviceClassCmd)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	e, err := GetResourceEntry(resource, "deviceClass1")
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(e.Features["name"], "deviceClass_name") {
		t.Error(e)
		return
	}

	result, err := GetOrderedListForUserOrGroup(resource, "testOwner", []string{}, "r", "3", "0", "name", true)
	if err != nil {
		t.Error(err)
		return
	}
	if len(result) != 1 {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[0]["name"], "deviceClass_name") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[0]["image"], "image1") {
		t.Error(result)
		return
	}
}

func getDeviceClassTestObj(id string, obj map[string]interface{}) (msg []byte, command CommandWrapper, err error) {
	text := `{
		"command": "PUT",
		"id": "%s",
		"owner": "testOwner",
		"device_class": %s
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
