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
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	kafka2 "github.com/SENERGY-Platform/permission-search/lib/worker/kafka"
	"github.com/segmentio/kafka-go"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestRightsCommand(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.LoadConfig("./../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	config.FatalErrHandler = t.Fatal

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

	t.Run("create aspects", createTestAspects(ctx, config, "aspect1", "aaaa", "aspect2", "aspect3", "aspect4", "aspect5"))

	time.Sleep(10 * time.Second) //kafka latency

	t.Run("list owner", testRequestWithToken(config, testtoken, "GET", "/v3/resources/aspects?rights=a", nil, 200, []map[string]interface{}{
		getTestAspectResult("aaaa"),
		getTestAspectResult("aspect1"),
		getTestAspectResult("aspect2"),
		getTestAspectResult("aspect3"),
		getTestAspectResult("aspect4"),
		getTestAspectResult("aspect5"),
	}))

	t.Run("list admin", testRequestWithToken(config, admintoken, "GET", "/v3/resources/aspects?rights=a", nil, 200, []map[string]interface{}{
		getTestAspectResult("aaaa"),
		getTestAspectResult("aspect1"),
		getTestAspectResult("aspect2"),
		getTestAspectResult("aspect3"),
		getTestAspectResult("aspect4"),
		getTestAspectResult("aspect5"),
	}))

	t.Run("list secondOwner", testRequestWithToken(config, secondOwnerToken, "GET", "/v3/resources/aspects?rights=a", nil, 200, nil))

	t.Run("secondOwner may not change rights", testRequestWithToken(config, secondOwnerToken, "PUT", "/v3/administrate/rights/aspects/aspect1", model.ResourceRightsBase{
		UserRights: map[string]model.Right{
			secendOwnerTokenUser: {
				Read:         true,
				Write:        true,
				Execute:      true,
				Administrate: true,
			}},
		GroupRights: map[string]model.Right{},
	}, http.StatusForbidden, nil))

	t.Run("owner may change rights 1", testRequestWithToken(config, testtoken, "PUT", "/v3/administrate/rights/aspects/aspect1", model.ResourceRightsBase{
		UserRights: map[string]model.Right{
			testTokenUser: {
				Read:         true,
				Write:        true,
				Execute:      true,
				Administrate: true,
			}},
		GroupRights: map[string]model.Right{},
	}, http.StatusOK, nil))

	t.Run("owner may change rights 2", testRequestWithToken(config, testtoken, "PUT", "/v3/administrate/rights/aspects/aspect2", model.ResourceRightsBase{
		UserRights: map[string]model.Right{
			testTokenUser: {
				Read:         true,
				Write:        true,
				Execute:      true,
				Administrate: true,
			}},
		GroupRights: map[string]model.Right{},
	}, http.StatusOK, nil))

	t.Run("admin may still change rights", testRequestWithToken(config, admintoken, "PUT", "/v3/administrate/rights/aspects/aspect1", model.ResourceRightsBase{
		UserRights: map[string]model.Right{
			testTokenUser: {
				Read:         true,
				Write:        true,
				Execute:      true,
				Administrate: true,
			},
			secendOwnerTokenUser: {
				Read:         true,
				Write:        true,
				Execute:      true,
				Administrate: true,
			}},
		GroupRights: map[string]model.Right{},
	}, http.StatusOK, nil))

	time.Sleep(10 * time.Second) //kafka latency

	t.Run("list owner after rights change", testRequestWithToken(config, testtoken, "GET", "/v3/resources/aspects?rights=a", nil, 200, []map[string]interface{}{
		getTestAspectResult("aaaa"),
		getTestAspectResultWithPermissionHolders("aspect1", []string{testTokenUser, secendOwnerTokenUser}, true),
		getTestAspectResult("aspect2"),
		getTestAspectResult("aspect3"),
		getTestAspectResult("aspect4"),
		getTestAspectResult("aspect5"),
	}))

	t.Run("list admin after rights change", testRequestWithToken(config, admintoken, "GET", "/v3/resources/aspects?rights=a", nil, 200, []map[string]interface{}{
		getTestAspectResult("aaaa"),
		getTestAspectResult("aspect3"),
		getTestAspectResult("aspect4"),
		getTestAspectResult("aspect5"),
	}))

	t.Run("list secondOwner after rights change", testRequestWithToken(config, secondOwnerToken, "GET", "/v3/resources/aspects?rights=a", nil, 200, []map[string]interface{}{
		getTestAspectResultWithPermissionHolders("aspect1", []string{testTokenUser, secendOwnerTokenUser}, true),
	}))

}

func TestRightsCommandKey(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := configuration.LoadConfig("./../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	config.FatalErrHandler = t.Fatal

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

	t.Run("without key", testRequestWithToken(config, admintoken, "PUT", "/v3/administrate/rights/aspects/aspect1", model.ResourceRightsBase{
		UserRights: map[string]model.Right{
			testTokenUser: {
				Read:         true,
				Write:        true,
				Execute:      true,
				Administrate: true,
			},
		},
		GroupRights: map[string]model.Right{},
	}, http.StatusOK, nil))

	t.Run("without key", testRequestWithToken(config, admintoken, "PUT", "/v3/administrate/rights/aspects/aspect2?key="+url.QueryEscape("prefix/suffix:aspect2"), model.ResourceRightsBase{
		UserRights: map[string]model.Right{
			testTokenUser: {
				Read:         true,
				Write:        true,
				Execute:      true,
				Administrate: true,
			},
		},
		GroupRights: map[string]model.Right{},
	}, http.StatusOK, nil))

	t.Run("consume", func(t *testing.T) {
		broker, err := kafka2.GetBroker(config.KafkaUrl)
		if err != nil {
			t.Error(err)
			return
		}
		r := kafka.NewReader(kafka.ReaderConfig{
			CommitInterval: 0, //synchronous commits
			Brokers:        broker,
			GroupID:        "test",
			Topic:          "aspects",
			MaxWait:        1 * time.Second,
			Logger:         log.New(ioutil.Discard, "", 0),
			ErrorLogger:    log.New(os.Stdout, "[KAFKA-ERROR] ", log.Default().Flags()),
		})

		consumerCtx, consumerCancel := context.WithTimeout(ctx, 10*time.Second)
		count := 0

		keys := map[string]int{}

		go func() {
			defer r.Close()
			for {
				select {
				case <-consumerCtx.Done():
					return
				default:
					m, err := r.FetchMessage(ctx)
					if err == io.EOF || err == context.Canceled {
						return
					}
					if err != nil {
						t.Error(err)
						return
					}

					keys[string(m.Key)] = keys[string(m.Key)] + 1

					err = r.CommitMessages(ctx, m)
					if err != nil {
						t.Error(err)
						return
					}
					count++
					if count >= 2 {
						consumerCancel()
					}
				}
			}
		}()

		<-consumerCtx.Done()

		if !reflect.DeepEqual(keys, map[string]int{
			"prefix/suffix:aspect2": 1,
			"aspect1/rights":        1,
		}) {
			t.Error(keys)
		}
	})

}
