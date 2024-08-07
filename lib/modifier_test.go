/*
 * Copyright 2022 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package lib

import (
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/query/modifier"
	k "github.com/SENERGY-Platform/permission-search/lib/worker/kafka"
	"log"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestResultModifiers(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.LoadConfig("./../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	config.LogDeprecatedCallsToFile = ""
	config.FatalErrHandler = func(v ...interface{}) {
		log.Println("TEST-ERROR:", v)
		t.Log(v...)
	}

	config.OpenSearchInsecureSkipVerify = true
	config.OpenSearchUsername = "admin"
	config.OpenSearchPassword = "01J1iEnT#>kE"

	t.Run("start dependency containers", func(t *testing.T) {
		_, ip, err := OpenSearch(ctx, wg)
		if err != nil {
			t.Error(err)
			return
		}
		config.OpenSearchUrls = "https://" + ip + ":9200"

		_, zkIp, err := Zookeeper(ctx, wg)
		if err != nil {
			t.Error(err)
			return
		}
		config.KafkaUrl = zkIp + ":2181"

		//kafka
		config.KafkaUrl, err = Kafka(ctx, wg, config.KafkaUrl)
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("start server", func(t *testing.T) {
		freePort, err := GetFreePort()
		if err != nil {
			t.Error(err)
			return
		}
		config.ServerPort = strconv.Itoa(freePort)
		err = Start(ctx, config, Standalone)
		if err != nil {
			t.Error(err)
			return
		}
	})

	dtId := "urn:infai:ses:device-type:bce1b3a2-8f46-44e7-9077-6d0a721be0a2"
	serviceGroupKey := "a8ee3b1c-4cda-4f0d-9f55-4ef4882ce0af"
	serviceGroupName := "Left Switch"
	t.Run("create device-type", createTestDeviceTypeWithServiceGroups(ctx, config, dtId, "device-type-name", "device-class-id", []map[string]interface{}{
		{
			"description": "",
			"key":         serviceGroupKey,
			"name":        serviceGroupName,
		},
		{
			"description": "",
			"key":         "c1b4e64f-5098-4d5b-92e5-c000eae64b33",
			"name":        "Right Switch",
		},
	}))

	dId := "urn:infai:ses:device:a8488d92-891d-4909-88c7-6fd9a2adfa10"
	dName := "device-name"

	idModifier := modifier.Seperator + modifier.EncodeModifierParameter(map[string][]string{"service_group_selection": {serviceGroupKey}})
	dtIdModified := dtId + idModifier
	dIdWithModify := dId + idModifier

	dNameModify := dName + " " + serviceGroupName

	t.Run("create device", createTestDeviceWithDeviceType(ctx, config, dId, dName, dtId))

	time.Sleep(10 * time.Second) //kafka latency

	t.Run("list", testRequest(config, "GET", "/v3/resources/devices", nil, 200, []map[string]interface{}{
		getTestDeviceResultWithDeviceTypeIdAndName(dId, dName, dtId),
	}))

	t.Run("search", testRequest(config, "GET", "/v3/resources/devices?limit=3&search=device", nil, 200, []map[string]interface{}{
		getTestDeviceResultWithDeviceTypeIdAndName(dId, dName, dtId),
	}))

	t.Run("ids unmodified", testRequest(config, "GET", "/v3/resources/devices?ids="+url.QueryEscape(dId), nil, 200, []map[string]interface{}{
		getTestDeviceResultWithDeviceTypeIdAndName(dId, dName, dtId),
	}))

	t.Run("ids modified", testRequest(config, "GET", "/v3/resources/devices?ids="+url.QueryEscape(dIdWithModify), nil, 200, []map[string]interface{}{
		getTestDeviceResultWithDeviceTypeIdAndName(dIdWithModify, dNameModify, dtIdModified),
	}))
	t.Run("ids modified not encoded", testRequest(config, "GET", "/v3/resources/devices?ids="+dIdWithModify, nil, 200, []map[string]interface{}{
		getTestDeviceResultWithDeviceTypeIdAndName(dIdWithModify, dNameModify, dtIdModified),
	}))
	t.Run("ids both", testRequest(config, "GET", "/v3/resources/devices?ids="+url.QueryEscape(dId)+","+url.QueryEscape(dIdWithModify), nil, 200, []map[string]interface{}{
		getTestDeviceResultWithDeviceTypeIdAndName(dId, dName, dtId),
		getTestDeviceResultWithDeviceTypeIdAndName(dIdWithModify, dNameModify, dtIdModified),
	}))
	t.Run("ids both not encoded", testRequest(config, "GET", "/v3/resources/devices?ids="+dId+","+dIdWithModify, nil, 200, []map[string]interface{}{
		getTestDeviceResultWithDeviceTypeIdAndName(dId, dName, dtId),
		getTestDeviceResultWithDeviceTypeIdAndName(dIdWithModify, dNameModify, dtIdModified),
	}))

	t.Run("access true", testRequest(config, "GET", "/v3/resources/devices/"+url.PathEscape(dIdWithModify)+"/access", nil, 200, true))
	t.Run("access true not encoded", testRequest(config, "GET", "/v3/resources/devices/"+dIdWithModify+"/access", nil, 200, true))

	t.Run("rights", testRequest(config, "GET", "/v3/administrate/rights/devices/"+url.PathEscape(dIdWithModify), nil, 200, nil))
	t.Run("rights not encoded", testRequest(config, "GET", "/v3/administrate/rights/devices/"+dIdWithModify, nil, 200, nil))

	expectedRights := map[string]interface{}{}
	json.Unmarshal([]byte(`{"creator":"testOwner","features":{"attributes":null,"device_type_id":"urn:infai:ses:device-type:bce1b3a2-8f46-44e7-9077-6d0a721be0a2$service_group_selection=a8ee3b1c-4cda-4f0d-9f55-4ef4882ce0af","display_name":"device-name Left Switch","local_id":null,"name":"device-name Left Switch","nickname":null,"owner_id":null},"group_rights":{"admin":{"administrate":true,"execute":true,"read":true,"write":true}},"resource_id":"urn:infai:ses:device:a8488d92-891d-4909-88c7-6fd9a2adfa10$service_group_selection=a8ee3b1c-4cda-4f0d-9f55-4ef4882ce0af","user_rights":{"testOwner":{"administrate":true,"execute":true,"read":true,"write":true}}}`),
		&expectedRights)

	t.Run("rights value", testRequest(config, "GET", "/v3/administrate/rights/devices/"+url.PathEscape(dIdWithModify), nil, 200, expectedRights))

	t.Run("query search", testRequest(config, "POST", "/v3/query", model.QueryMessage{
		Resource: "devices",
		Find: &model.QueryFind{
			QueryListCommons: model.QueryListCommons{
				Limit:    3,
				Offset:   0,
				SortBy:   "name",
				SortDesc: true,
			},
			Search: "device",
		},
	}, 200, []map[string]interface{}{
		getTestDeviceResultWithDeviceTypeIdAndName(dId, dName, dtId),
	}))

	t.Run("query ids", testRequest(config, "POST", "/v3/query", model.QueryMessage{
		Resource: "devices",
		ListIds: &model.QueryListIds{
			QueryListCommons: model.QueryListCommons{
				Limit:    3,
				Offset:   0,
				SortBy:   "name",
				SortDesc: true,
			},
			Ids: []string{dId, dIdWithModify},
		},
	}, 200, []map[string]interface{}{
		getTestDeviceResultWithDeviceTypeIdAndName(dId, dName, dtId),
		getTestDeviceResultWithDeviceTypeIdAndName(dIdWithModify, dNameModify, dtIdModified),
	}))

	t.Run("query check ids", testRequest(config, "POST", "/v3/query", model.QueryMessage{
		Resource: "devices",
		CheckIds: &model.QueryCheckIds{
			Ids: []string{dId, dIdWithModify},
		},
	}, 200, map[string]bool{
		dId:           true,
		dIdWithModify: true,
	}))

	t.Run("add modify to query filter result", testRequest(config, "POST", "/v3/query", model.QueryMessage{
		Resource: "devices",
		Find: &model.QueryFind{
			QueryListCommons: model.QueryListCommons{
				AddIdModifier: map[string][]string{
					"service_group_selection": {serviceGroupKey},
				},
			},
			Filter: &model.Selection{
				Condition: model.ConditionConfig{
					Feature:   "features.device_type_id",
					Operation: model.QueryEqualOperation,
					Value:     dtId,
				},
			},
		},
	}, 200, []map[string]interface{}{
		getTestDeviceResultWithDeviceTypeIdAndName(dIdWithModify, dNameModify, dtIdModified),
	}))

	t.Run("add modify to filter result", testRequest(config, "GET", "/v3/resources/devices?filter="+url.PathEscape("device_type_id:"+dtId)+"&add_id_modifier="+url.QueryEscape(modifier.EncodeModifierParameter(map[string][]string{"service_group_selection": {serviceGroupKey}})),
		nil,
		200,
		[]map[string]interface{}{
			getTestDeviceResultWithDeviceTypeIdAndName(dIdWithModify, dNameModify, dtIdModified),
		}))

}

func createTestDeviceWithDeviceType(ctx context.Context, config configuration.Config, id string, name string, deviceTypeId string) func(t *testing.T) {
	return func(t *testing.T) {
		p, err := k.NewProducer(ctx, config.KafkaUrl, "devices", true)
		if err != nil {
			t.Error(err)
			return
		}
		deviceMsg, deviceCmd, err := getDeviceTestObj(id, map[string]interface{}{
			"name":           name,
			"device_type_id": deviceTypeId,
		})
		if err != nil {
			t.Error(err)
			return
		}
		err = p.Produce(deviceCmd.Id, deviceMsg)
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func createTestDeviceTypeWithServiceGroups(ctx context.Context, config configuration.Config, id string, name string, deviceClassId string, serviceGroups interface{}) func(t *testing.T) {
	return func(t *testing.T) {
		p, err := k.NewProducer(ctx, config.KafkaUrl, "device-types", true)
		if err != nil {
			t.Error(err)
			return
		}
		msg, cmd := getDtTestObj(id, map[string]interface{}{
			"id":              id,
			"name":            name,
			"device_class_id": deviceClassId,
			"service_groups":  serviceGroups,
		})
		err = p.Produce(cmd.Id, msg)
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func getTestDeviceResultWithDeviceTypeIdAndName(id string, name string, deviceTypeId string) (result map[string]interface{}) {
	result = map[string]interface{}{
		"creator":        "testOwner",
		"id":             id,
		"name":           name,
		"nickname":       nil,
		"display_name":   name,
		"attributes":     nil,
		"device_type_id": deviceTypeId,
		"local_id":       nil,
		"owner_id":       nil,
		"permissions": map[string]bool{
			"a": true,
			"r": true,
			"w": true,
			"x": true,
		},
		"permission_holders": map[string][]string{
			"admin_users":   {"testOwner"},
			"execute_users": {"testOwner"},
			"read_users":    {"testOwner"},
			"write_users":   {"testOwner"},
		},
		"shared": false,
	}

	return result
}
