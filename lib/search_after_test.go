package lib

import (
	"context"
	"fmt"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/worker"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestSearchAfter(t *testing.T) {
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

	var w *worker.Worker
	t.Run("start server", func(t *testing.T) {
		freePort, err := GetFreePort()
		if err != nil {
			t.Error(err)
			return
		}
		config.ServerPort = strconv.Itoa(freePort)
		_, w, err = StartGetComponents(ctx, config, Standalone)
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("create entries", func(t *testing.T) {
		handler := w.GetResourceCommandHandler("aspects")
		for i := 1; i <= 20000; i++ {
			id := fmt.Sprintf("%06d", i)
			log.Println("create", id)
			aspectMsg, _, err := getAspectTestObj(id, map[string]interface{}{
				"name":     id + "_name",
				"rdf_type": "aspect_type",
			})
			if err != nil {
				t.Error(err)
				return
			}
			err = handler(aspectMsg)
			if err != nil {
				t.Error(err)
				return
			}
		}
	})

	time.Sleep(1 * time.Second)

	t.Run("access true --> last entry exists", testRequest(config, "GET", "/v3/resources/aspects/020000/access", nil, 200, true))

	t.Run("list offset 9900", testRequest(config, "GET", "/v3/resources/aspects?limit=3&offset=9900", nil, 200, []map[string]interface{}{
		getTestAspectResult("009901"),
		getTestAspectResult("009902"),
		getTestAspectResult("009903"),
	}))

	t.Run("list after 009900", testRequest(config, "GET", "/v3/resources/aspects?limit=3&after.id=009900&after.sort_field_value="+url.QueryEscape(`"009900_name"`), nil, 200, []map[string]interface{}{
		getTestAspectResult("009901"),
		getTestAspectResult("009902"),
		getTestAspectResult("009903"),
	}))

	t.Run("list offset 19900", testRequest(config, "GET", "/v3/resources/aspects?limit=3&offset=19900", nil, http.StatusBadRequest, nil))

	t.Run("list after 019900", testRequest(config, "GET", "/v3/resources/aspects?limit=3&after.id=019900&after.sort_field_value="+url.QueryEscape(`"019900_name"`), nil, 200, []map[string]interface{}{
		getTestAspectResult("019901"),
		getTestAspectResult("019902"),
		getTestAspectResult("019903"),
	}))
}