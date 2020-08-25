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

func TestReceiveFunction(t *testing.T) {
	resource := "functions"
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
	functionMsg, functionCmd, err := getFunctionTestObj("function1", map[string]interface{}{
		"name":       "function_name",
		"rdf_type":   "function_type",
		"concept_id": "concept_id_1",
	})
	if err != nil {
		t.Error(err)
		return
	}
	err = UpdateFeatures(resource, functionMsg, functionCmd)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(2 * time.Second)

	e, err := GetResourceEntry(resource, "function1")
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(e.Features["name"], "function_name") {
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
	if !reflect.DeepEqual(result[0]["name"], "function_name") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[0]["rdf_type"], "function_type") {
		t.Error(result)
		return
	}
	if !reflect.DeepEqual(result[0]["concept_id"], "concept_id_1") {
		t.Error(result)
		return
	}
}

func getFunctionTestObj(id string, obj map[string]interface{}) (msg []byte, command CommandWrapper, err error) {
	text := `{
		"command": "PUT",
		"id": "%s",
		"owner": "testOwner",
		"function": %s
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
