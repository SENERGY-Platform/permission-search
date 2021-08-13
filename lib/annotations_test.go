package lib

import (
	"context"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	k "github.com/SENERGY-Platform/permission-search/lib/worker/kafka"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestAnnotations(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.LoadConfig("./../config.json")
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("start dependency containers", func(t *testing.T) {
		port, _, err := elasticsearch(ctx, wg)
		if err != nil {
			t.Error(err)
			return
		}
		config.ElasticUrl = "http://localhost:" + port

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

	t.Run("create devices", createTestDevices(ctx, config, "device1", "device2", "device3", "device4", "device5"))

	t.Run("send connection state for device2 = connected", sendTestConnectionState(ctx, config, "device2", true))
	t.Run("send connection state for device3 = connected", sendTestConnectionState(ctx, config, "device3", true))
	t.Run("send connection state for device4 = disconnected", sendTestConnectionState(ctx, config, "device4", false))
	t.Run("send connection state for device2 = disconnected", sendTestConnectionState(ctx, config, "device2", false))

	time.Sleep(10 * time.Second) //kafka latency

	trueVar := true
	falseVar := false
	t.Run("query connected", testRequest(config, "POST", "/v3/query", model.QueryMessage{
		Resource: "devices",
		Find: &model.QueryFind{
			Filter: &model.Selection{
				Condition: model.ConditionConfig{
					Feature:   "annotations.connected",
					Operation: "==",
					Value:     true,
				},
			},
		},
	}, 200, []map[string]interface{}{
		getTestDeviceResult("device3", &trueVar),
	}))

	t.Run("query disconnected", testRequest(config, "POST", "/v3/query", model.QueryMessage{
		Resource: "devices",
		Find: &model.QueryFind{
			Filter: &model.Selection{
				Condition: model.ConditionConfig{
					Feature:   "annotations.connected",
					Operation: "==",
					Value:     false,
				},
			},
		},
	}, 200, []map[string]interface{}{
		getTestDeviceResult("device2", &falseVar),
		getTestDeviceResult("device4", &falseVar),
	}))

	t.Run("query unknown", testRequest(config, "POST", "/v3/query", model.QueryMessage{
		Resource: "devices",
		Find: &model.QueryFind{
			Filter: &model.Selection{
				Condition: model.ConditionConfig{
					Feature:   "annotations.connected",
					Operation: "==",
					Value:     nil,
				},
			},
		},
	}, 200, []map[string]interface{}{
		getTestDeviceResult("device1", nil),
		getTestDeviceResult("device5", nil),
	}))

}

func sendTestConnectionState(ctx context.Context, config configuration.Config, id string, connected bool) func(t *testing.T) {
	return func(t *testing.T) {
		p, err := k.NewProducer(ctx, config.KafkaUrl, "device_log", true)
		if err != nil {
			t.Error(err)
			return
		}
		err = p.Produce(id, []byte(`{"id": "`+id+`", "connected": `+strconv.FormatBool(connected)+`}`))
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func createTestDevices(ctx context.Context, config configuration.Config, ids ...string) func(t *testing.T) {
	return func(t *testing.T) {
		p, err := k.NewProducer(ctx, config.KafkaUrl, "devices", true)
		if err != nil {
			t.Error(err)
			return
		}
		for _, id := range ids {
			t.Run("create "+id, createTestDevice(p, id))
		}
	}
}

func createTestDevice(p *k.Producer, id string) func(t *testing.T) {
	return func(t *testing.T) {
		aspectMsg, aspectCmd, err := getDeviceTestObj(id, map[string]interface{}{
			"name": id + "_name",
		})
		if err != nil {
			t.Error(err)
			return
		}
		err = p.Produce(aspectCmd.Id, aspectMsg)
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func getTestDeviceResult(id string, connected *bool) (result map[string]interface{}) {
	result = map[string]interface{}{
		"creator":        "testOwner",
		"id":             id,
		"name":           id + "_name",
		"attributes":     nil,
		"device_type_id": nil,
		"local_id":       nil,
		"permissions": map[string]bool{
			"a": true,
			"r": true,
			"w": true,
			"x": true,
		},
		"shared": false,
	}
	if connected != nil {
		result["annotations"] = map[string]interface{}{
			"connected": *connected,
		}
	}
	return result
}
