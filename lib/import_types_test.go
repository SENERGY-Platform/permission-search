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

func TestImportTypes(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, q, w, err := getTestEnv(ctx, wg)
	if err != nil {
		fmt.Println(err)
		return
	}

	resource := "import-types"
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

	msg, cmd, err := getImportTypeObj("g1", map[string]interface{}{
		"name":            "name",
		"description":     "description",
		"image":           "image",
		"default_restart": true,
		"configs": []interface{}{
			map[string]interface{}{
				"name":          "c1",
				"description":   "description c1",
				"type":          "https://schema.org/Text",
				"default_value": "val",
			},
			map[string]interface{}{
				"name":          "c2",
				"description":   "description c2",
				"type":          "https://schema.org/Float",
				"default_value": 15.3,
			},
		},
		"aspect_ids": []string{
			"a1",
			"a2",
		},
		"output": map[string]interface{}{
			"name":              "output",
			"type":              "https://schema.org/StructuredValue",
			"characteristic_id": "characteristic",
			"sub_content_variables": []interface{}{

				map[string]interface{}{
					"name":              "sub",
					"type":              "https://schema.org/Text",
					"characteristic_id": "text-characteristic",
				},
			},
		},
		"function_ids": []string{
			"f1",
			"f2",
		},
		"aspect_functions": []string{
			"a1_f1",
			"a1_f2",
			"a2_f1",
			"a2_f2",
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
	if !reflect.DeepEqual(e.Features["name"], "name") {
		t.Error(e)
		return
	}

	expected := []map[string]interface{}{{
		"id":              "g1",
		"name":            "name",
		"description":     "description",
		"image":           "image",
		"default_restart": true,
		"creator":         "testOwner",
		"function_ids": []interface{}{
			"f1",
			"f2",
		},
		"aspect_functions": []interface{}{
			"a1_f1",
			"a1_f2",
			"a2_f1",
			"a2_f2",
		},
		"permissions": map[string]bool{
			"a": true,
			"r": true,
			"w": true,
			"x": true,
		},
		"aspect_ids": []interface{}{
			"a1",
			"a2",
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

	result, err = q.SelectByFieldOrdered(
		resource,
		"aspect_functions",
		"a1_f1",
		"testOwner",
		[]string{"user"},
		"r",
		"3",
		"0",
		"name",
		true)

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

func getImportTypeObj(id string, obj map[string]interface{}) (msg []byte, command model.CommandWrapper, err error) {
	text := `{
		"command": "PUT",
		"id": "%s",
		"owner": "testOwner",
		"import_type": %s
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
