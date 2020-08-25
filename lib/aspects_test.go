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

func TestReceiveAspect(t *testing.T) {
	resource := "aspects"
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
	aspectMsg, aspectCmd, err := getAspectTestObj("aspect1", map[string]interface{}{
		"name":     "aspect_name",
		"rdf_type": "aspect_type",
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = UpdateFeatures(resource, aspectMsg, aspectCmd)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	e, err := GetResourceEntry(resource, "aspect1")
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(e.Features["name"], "aspect_name") {
		t.Error(e)
		return
	}

	result, err := GetOrderedListForUserOrGroup(resource, "testOwner", []string{"user"}, "r", "3", "0", "name", true)
	if err != nil {
		t.Error(err)
		return
	}
	if len(result) != 1 {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[0]["name"], "aspect_name") {
		t.Error(result)
		return
	}
}

func getAspectTestObj(id string, obj map[string]interface{}) (msg []byte, command CommandWrapper, err error) {
	text := `{
		"command": "PUT",
		"id": "%s",
		"owner": "testOwner",
		"aspect": %s
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
