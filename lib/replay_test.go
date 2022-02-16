package lib

import (
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/replay"
	k "github.com/SENERGY-Platform/permission-search/lib/worker/kafka"
	"sync"
	"testing"
	"time"
)

func TestReplay(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var initial = replay.DefaultBatchSize
	replay.DefaultBatchSize = 3
	defer func() {
		replay.DefaultBatchSize = initial
	}()

	config, err := configuration.LoadConfig("./../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	config.FatalErrHandler = t.Fatal
	config.Debug = true

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

	t.Run("start server", startTestServer(config, confV1, ctx))

	t.Run("add elements", func(t *testing.T) {
		p, err := k.NewProducer(ctx, config.KafkaUrl, "device-types", true)
		if err != nil {
			t.Error(err)
			return
		}
		t.Run("create dt1", createTestDeviceType(p, "dt1", "n1", "dc1"))
		t.Run("create dt2", createTestDeviceType(p, "dt2", "n2", "dc2"))
		t.Run("create dt3", createTestDeviceType(p, "dt3", "n3", "dc2"))
		t.Run("create dt4", createTestDeviceType(p, "dt4", "n4", "dc2"))
		t.Run("create dt5", createTestDeviceType(p, "dt5", "n5", "dc2"))
		t.Run("create dt6", createTestDeviceType(p, "dt6", "n6", "dc2"))
		t.Run("create dt7", createTestDeviceType(p, "dt7", "n7", "dc2"))
		t.Run("create dt8", createTestDeviceType(p, "dt8", "n8", "dc2"))
		t.Run("create dt9", createTestDeviceType(p, "dt9", "n9", "dc2"))
		t.Run("create dt10", createTestDeviceType(p, "dt10", "n10", "dc2"))
		t.Run("create dt11", createTestDeviceType(p, "dt11", "n11", "dc2"))
	})

	time.Sleep(5 * time.Second)

	t.Run("add user right", testReplayAddRights(config))

	time.Sleep(5 * time.Second)

	t.Run("replay default", func(t *testing.T) {
		replay.ReplayPermissions(config, []string{})
	})

	t.Run("replay do", func(t *testing.T) {
		replay.ReplayPermissions(config, []string{"do"})
	})

	t.Run("replay device-types", func(t *testing.T) {
		replay.ReplayPermissions(config, []string{"device-types"})
	})

	t.Run("replay do device-types", func(t *testing.T) {
		replay.ReplayPermissions(config, []string{"do", "device-types"})
	})
}

func testReplayAddRights(config configuration.Config) func(t *testing.T) {
	return func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		p, err := k.NewProducer(ctx, config.KafkaUrl, "permissions", true)
		if err != nil {
			t.Error(err)
			return
		}
		t.Run("user2 dt1", sendTestPermission(p, model.PermCommandMsg{
			Command:  "PUT",
			Kind:     "device-types",
			Resource: "dt1",
			User:     "user2",
			Right:    "rx",
		}))
		t.Run("moderator dt2", sendTestPermission(p, model.PermCommandMsg{
			Command:  "PUT",
			Kind:     "device-types",
			Resource: "dt2",
			Group:    "moderator",
			Right:    "rwa",
		}))
	}
}

func sendTestPermission(p *k.Producer, msg model.PermCommandMsg) func(t *testing.T) {
	return func(t *testing.T) {
		pl, err := json.Marshal(msg)
		if err != nil {
			t.Error(err)
			return
		}
		err = p.Produce(msg.Resource, pl)
		if err != nil {
			t.Error(err)
			return
		}
	}
}
