/*
 * Copyright 2018 InfAI (CC SES)
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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/permission-search/lib/api"
	"github.com/SENERGY-Platform/permission-search/lib/auth"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/query"
	"github.com/SENERGY-Platform/permission-search/lib/worker"
	k "github.com/SENERGY-Platform/permission-search/lib/worker/kafka"
	"github.com/segmentio/kafka-go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/wvanbergen/kazoo-go"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	elastic "github.com/olivere/elastic/v7"
)

func getDtTestObj(id string, dt map[string]interface{}) (msg []byte, command model.CommandWrapper) {
	text := `{
		"command": "PUT",
		"id": "%s",
		"owner": "testOwner",
		"device_type": %s
	}`
	dtStr, err := json.Marshal(dt)
	if err != nil {
		fmt.Println(err)
		return
	}
	msg = []byte(fmt.Sprintf(text, id, string(dtStr)))
	err = json.Unmarshal(msg, &command)
	if err != nil {
		fmt.Println(err)
		return
	}
	return
}

func initDb(config configuration.Config, worker *worker.Worker) {
	config.ElasticRetry = 3
	test, testCmd := getDtTestObj("test", map[string]interface{}{
		"name":        "test",
		"description": "something",
		"maintenance": []string{"something", "onotherthing"},
		"services":    []map[string]interface{}{{"id": "serviceTest1"}, {"id": "serviceTest2"}},
		"vendor":      map[string]interface{}{"name": "vendor"},
	})
	foo1, foo1Cmd := getDtTestObj("foo1", map[string]interface{}{
		"name":        "foo1",
		"description": "foo1Desc",
		"maintenance": []string{},
		"services":    []map[string]interface{}{{"id": "foo1Service"}},
		"vendor":      map[string]interface{}{"name": "foo1Vendor"},
	})
	foo2, foo2Cmd := getDtTestObj("foo2", map[string]interface{}{
		"name":        "foo2",
		"description": "foo2Desc",
		"maintenance": []string{},
		"services":    []map[string]interface{}{{"id": "foo2Service"}},
		"vendor":      map[string]interface{}{"name": "foo2Vendor"},
	})
	bar, barCmd := getDtTestObj("test", map[string]interface{}{
		"name":        "test",
		"description": "changedDesc",
		"maintenance": []string{"something", "different"},
		"services":    []map[string]interface{}{{"id": "serviceTest1"}, {"id": "serviceTest3"}},
		"vendor":      map[string]interface{}{"name": "chengedvendor"},
	})
	//ZWay-SwitchMultilevel
	zway, zwayCmd := getDtTestObj("zway", map[string]interface{}{
		"name":        "ZWay-SwitchMultilevel",
		"description": "desc",
		"maintenance": []string{},
		"services":    []map[string]interface{}{},
		"vendor":      map[string]interface{}{"name": "vendor"},
	})

	_, err := worker.GetClient().DeleteByQuery("device-types").Query(elastic.NewMatchAllQuery()).Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = worker.GetClient().Flush().Index("device-types").Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	err = worker.UpdateFeatures("device-types", test, testCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = worker.UpdateFeatures("device-types", foo1, foo1Cmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = worker.UpdateFeatures("device-types", foo2, foo2Cmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = worker.UpdateFeatures("device-types", bar, barCmd)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = worker.UpdateFeatures("device-types", zway, zwayCmd)
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = worker.GetClient().Flush().Index("device-types").Do(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
}

func elasticsearch(ctx context.Context, wg *sync.WaitGroup) (hostPort string, ipAddress string, err error) {
	log.Println("start elasticsearch")
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "docker.elastic.co/elasticsearch/elasticsearch:7.6.1",
			Env: map[string]string{
				"discovery.type": "single-node",
				"path.data":      "/opt/elasticsearch/volatile/data",
				"path.logs":      "/opt/elasticsearch/volatile/logs",
			},
			Tmpfs: map[string]string{
				"/opt/elasticsearch/volatile/data": "rw",
				"/opt/elasticsearch/volatile/logs": "rw",
				"/tmp":                             "rw",
			},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("9200/tcp"),
				wait.ForNop(waitretry(1*time.Minute, func(ctx context.Context, target wait.StrategyTarget) error {
					host, err := target.Host(ctx)
					if err != nil {
						log.Println("host", err)
						return err
					}
					port, err := target.MappedPort(ctx, "9200/tcp")
					if err != nil {
						log.Println("port", err)
						return err
					}
					return tryElasticSearchConnection(host, port.Port())
				}))),
			ExposedPorts:    []string{"9200/tcp"},
			AlwaysPullImage: true,
		},
		Started: true,
	})
	if err != nil {
		return "", "", err
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		log.Println("DEBUG: remove container elasticsearch", c.Terminate(context.Background()))
	}()

	ipAddress, err = c.ContainerIP(ctx)
	if err != nil {
		return "", "", err
	}
	temp, err := c.MappedPort(ctx, "9200/tcp")
	if err != nil {
		return "", "", err
	}
	hostPort = temp.Port()

	err = tryElasticSearchConnection("localhost", hostPort)
	if err != nil {
		log.Println("ERROR: tryElasticSearchConnection(\"localhost\", hostPort)", err)
		return "", "", err
	}
	err = tryElasticSearchConnection(ipAddress, "9200")
	if err != nil {
		log.Println("ERROR: tryElasticSearchConnection(ipAddress, \"9200\")", err)
		return "", "", err
	}

	return hostPort, ipAddress, err
}

func tryElasticSearchConnection(ip string, port string) error {
	log.Println("try elastic connection to ", ip, port, "...")
	resp, err := http.Get("http://" + ip + ":" + port + "/_cluster/health?wait_for_status=green&timeout=50s")
	if err != nil {
		log.Println("ERROR:", err)
		return err
	}
	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		err = errors.New("unexpected status code " + resp.Status)
		log.Println("ERROR:", err, "\n", string(temp))
		return err
	}
	result := map[string]interface{}{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		log.Println("ERROR:", err)
		return err
	}
	if result["number_of_nodes"] != float64(1) || result["status"] != "green" {
		err = fmt.Errorf("%#v", result)
		log.Println("ERROR:", err)
		return err
	}
	return err
}

func getTestEnv(ctx context.Context, wg *sync.WaitGroup, t *testing.T) (config configuration.Config, q *query.Query, w *worker.Worker, err error) {
	config, err = configuration.LoadConfig("./../config.json")
	if err != nil {
		return config, q, w, err
	}
	config.LogDeprecatedCallsToFile = ""
	if t != nil {
		config.FatalErrHandler = func(v ...interface{}) {
			log.Println("TEST-ERROR:", v)
			t.Log(v...)
		}
	} else {
		config.FatalErrHandler = func(v ...interface{}) {
			log.Println(v...)
			return
		}
	}
	port, _, err := elasticsearch(ctx, wg)
	if err != nil {
		return config, q, w, err
	}
	config.ElasticUrl = "http://localhost:" + port
	q, err = query.New(config)
	if err != nil {
		return config, q, w, err
	}
	w, err = worker.New(ctx, config, q)
	if err != nil {
		return config, q, w, err
	}
	return
}

func getTestEnvWithApi(ctx context.Context, wg *sync.WaitGroup, t *testing.T) (config configuration.Config, q *query.Query, w *worker.Worker, err error) {
	config, q, w, err = getTestEnv(ctx, wg, t)
	if err != nil {
		return config, q, w, err
	}
	handler, err := api.GetRouter(config, q, nil)
	if err != nil {
		return config, q, w, err
	}
	server := httptest.NewServer(handler)
	serverUrl, err := url.Parse(server.URL)
	if err != nil {
		return config, q, w, err
	}
	wg.Add(1)
	go func() {
		<-ctx.Done()
		server.Close()
		wg.Done()
	}()
	config.ServerPort = serverUrl.Port()
	return
}

const testTokenUser = "testOwner"
const testtoken = `Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiIwOGM0N2E4OC0yYzc5LTQyMGYtODEwNC02NWJkOWViYmU0MWUiLCJleHAiOjE1NDY1MDcyMzMsIm5iZiI6MCwiaWF0IjoxNTQ2NTA3MTczLCJpc3MiOiJodHRwOi8vbG9jYWxob3N0OjgwMDEvYXV0aC9yZWFsbXMvbWFzdGVyIiwiYXVkIjoiZnJvbnRlbmQiLCJzdWIiOiJ0ZXN0T3duZXIiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJmcm9udGVuZCIsIm5vbmNlIjoiOTJjNDNjOTUtNzViMC00NmNmLTgwYWUtNDVkZDk3M2I0YjdmIiwiYXV0aF90aW1lIjoxNTQ2NTA3MDA5LCJzZXNzaW9uX3N0YXRlIjoiNWRmOTI4ZjQtMDhmMC00ZWI5LTliNjAtM2EwYWUyMmVmYzczIiwiYWNyIjoiMCIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJ1c2VyIl19LCJyZXNvdXJjZV9hY2Nlc3MiOnsibWFzdGVyLXJlYWxtIjp7InJvbGVzIjpbInZpZXctcmVhbG0iLCJ2aWV3LWlkZW50aXR5LXByb3ZpZGVycyIsIm1hbmFnZS1pZGVudGl0eS1wcm92aWRlcnMiLCJpbXBlcnNvbmF0aW9uIiwiY3JlYXRlLWNsaWVudCIsIm1hbmFnZS11c2VycyIsInF1ZXJ5LXJlYWxtcyIsInZpZXctYXV0aG9yaXphdGlvbiIsInF1ZXJ5LWNsaWVudHMiLCJxdWVyeS11c2VycyIsIm1hbmFnZS1ldmVudHMiLCJtYW5hZ2UtcmVhbG0iLCJ2aWV3LWV2ZW50cyIsInZpZXctdXNlcnMiLCJ2aWV3LWNsaWVudHMiLCJtYW5hZ2UtYXV0aG9yaXphdGlvbiIsIm1hbmFnZS1jbGllbnRzIiwicXVlcnktZ3JvdXBzIl19LCJhY2NvdW50Ijp7InJvbGVzIjpbIm1hbmFnZS1hY2NvdW50IiwibWFuYWdlLWFjY291bnQtbGlua3MiLCJ2aWV3LXByb2ZpbGUiXX19LCJyb2xlcyI6WyJ1c2VyIl19.ykpuOmlpzj75ecSI6cHbCATIeY4qpyut2hMc1a67Ycg`

const adminTokenUser = "admin"
const admintoken = `Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiIwOGM0N2E4OC0yYzc5LTQyMGYtODEwNC02NWJkOWViYmU0MWUiLCJleHAiOjE1NDY1MDcyMzMsIm5iZiI6MCwiaWF0IjoxNTQ2NTA3MTczLCJpc3MiOiJodHRwOi8vbG9jYWxob3N0OjgwMDEvYXV0aC9yZWFsbXMvbWFzdGVyIiwiYXVkIjoiZnJvbnRlbmQiLCJzdWIiOiJhZG1pbiIsInR5cCI6IkJlYXJlciIsImF6cCI6ImZyb250ZW5kIiwibm9uY2UiOiI5MmM0M2M5NS03NWIwLTQ2Y2YtODBhZS00NWRkOTczYjRiN2YiLCJhdXRoX3RpbWUiOjE1NDY1MDcwMDksInNlc3Npb25fc3RhdGUiOiI1ZGY5MjhmNC0wOGYwLTRlYjktOWI2MC0zYTBhZTIyZWZjNzMiLCJhY3IiOiIwIiwiYWxsb3dlZC1vcmlnaW5zIjpbIioiXSwicmVhbG1fYWNjZXNzIjp7InJvbGVzIjpbInVzZXIiLCJhZG1pbiJdfSwicmVzb3VyY2VfYWNjZXNzIjp7Im1hc3Rlci1yZWFsbSI6eyJyb2xlcyI6WyJ2aWV3LXJlYWxtIiwidmlldy1pZGVudGl0eS1wcm92aWRlcnMiLCJtYW5hZ2UtaWRlbnRpdHktcHJvdmlkZXJzIiwiaW1wZXJzb25hdGlvbiIsImNyZWF0ZS1jbGllbnQiLCJtYW5hZ2UtdXNlcnMiLCJxdWVyeS1yZWFsbXMiLCJ2aWV3LWF1dGhvcml6YXRpb24iLCJxdWVyeS1jbGllbnRzIiwicXVlcnktdXNlcnMiLCJtYW5hZ2UtZXZlbnRzIiwibWFuYWdlLXJlYWxtIiwidmlldy1ldmVudHMiLCJ2aWV3LXVzZXJzIiwidmlldy1jbGllbnRzIiwibWFuYWdlLWF1dGhvcml6YXRpb24iLCJtYW5hZ2UtY2xpZW50cyIsInF1ZXJ5LWdyb3VwcyJdfSwiYWNjb3VudCI6eyJyb2xlcyI6WyJtYW5hZ2UtYWNjb3VudCIsIm1hbmFnZS1hY2NvdW50LWxpbmtzIiwidmlldy1wcm9maWxlIl19fSwicm9sZXMiOlsidXNlciIsImFkbWluIl19.ggcFFFEsjwdfSzEFzmZt_m6W4IiSQub2FRhZVfWttDI`

const secendOwnerTokenUser = "secondOwner"
const secondOwnerToken = `Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiIwOGM0N2E4OC0yYzc5LTQyMGYtODEwNC02NWJkOWViYmU0MWUiLCJleHAiOjE1NDY1MDcyMzMsIm5iZiI6MCwiaWF0IjoxNTQ2NTA3MTczLCJpc3MiOiJodHRwOi8vbG9jYWxob3N0OjgwMDEvYXV0aC9yZWFsbXMvbWFzdGVyIiwiYXVkIjoiZnJvbnRlbmQiLCJzdWIiOiJzZWNvbmRPd25lciIsInR5cCI6IkJlYXJlciIsImF6cCI6ImZyb250ZW5kIiwibm9uY2UiOiI5MmM0M2M5NS03NWIwLTQ2Y2YtODBhZS00NWRkOTczYjRiN2YiLCJhdXRoX3RpbWUiOjE1NDY1MDcwMDksInNlc3Npb25fc3RhdGUiOiI1ZGY5MjhmNC0wOGYwLTRlYjktOWI2MC0zYTBhZTIyZWZjNzMiLCJhY3IiOiIwIiwiYWxsb3dlZC1vcmlnaW5zIjpbIioiXSwicmVhbG1fYWNjZXNzIjp7InJvbGVzIjpbInVzZXIiXX0sInJlc291cmNlX2FjY2VzcyI6eyJtYXN0ZXItcmVhbG0iOnsicm9sZXMiOlsidmlldy1yZWFsbSIsInZpZXctaWRlbnRpdHktcHJvdmlkZXJzIiwibWFuYWdlLWlkZW50aXR5LXByb3ZpZGVycyIsImltcGVyc29uYXRpb24iLCJjcmVhdGUtY2xpZW50IiwibWFuYWdlLXVzZXJzIiwicXVlcnktcmVhbG1zIiwidmlldy1hdXRob3JpemF0aW9uIiwicXVlcnktY2xpZW50cyIsInF1ZXJ5LXVzZXJzIiwibWFuYWdlLWV2ZW50cyIsIm1hbmFnZS1yZWFsbSIsInZpZXctZXZlbnRzIiwidmlldy11c2VycyIsInZpZXctY2xpZW50cyIsIm1hbmFnZS1hdXRob3JpemF0aW9uIiwibWFuYWdlLWNsaWVudHMiLCJxdWVyeS1ncm91cHMiXX0sImFjY291bnQiOnsicm9sZXMiOlsibWFuYWdlLWFjY291bnQiLCJtYW5hZ2UtYWNjb3VudC1saW5rcyIsInZpZXctcHJvZmlsZSJdfX0sInJvbGVzIjpbInVzZXIiXX0.cq8YeUuR0jSsXCEzp634fTzNbGkq_B8KbVrwBPgceJ4`

func Kafka(ctx context.Context, wg *sync.WaitGroup, zookeeperUrl string) (kafkaUrl string, err error) {
	kafkaport, err := GetFreePort()
	if err != nil {
		return kafkaUrl, err
	}
	provider, err := testcontainers.NewDockerProvider(testcontainers.DefaultNetwork("bridge"))
	if err != nil {
		return kafkaUrl, err
	}
	hostIp, err := provider.GetGatewayIP(ctx)
	if err != nil {
		return kafkaUrl, err
	}
	kafkaUrl = hostIp + ":" + strconv.Itoa(kafkaport)
	log.Println("host ip: ", hostIp)
	log.Println("host port: ", kafkaport)
	log.Println("kafkaUrl url: ", kafkaUrl)
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "bitnami/kafka:3.4.0-debian-11-r21",
			Tmpfs: map[string]string{},
			WaitingFor: wait.ForAll(
				//wait.ForLog("INFO Awaiting socket connections on"),
				wait.ForListeningPort("9092/tcp"),
			),
			ExposedPorts:    []string{"9092/tcp"},
			AlwaysPullImage: true,
			Env: map[string]string{
				"ALLOW_PLAINTEXT_LISTENER":             "yes",
				"KAFKA_LISTENERS":                      "OUTSIDE://:9092",
				"KAFKA_ADVERTISED_LISTENERS":           "OUTSIDE://" + kafkaUrl,
				"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP": "OUTSIDE:PLAINTEXT",
				"KAFKA_INTER_BROKER_LISTENER_NAME":     "OUTSIDE",
				"KAFKA_ZOOKEEPER_CONNECT":              zookeeperUrl,
			},
		},
		Started: true,
	})
	if err != nil {
		debug.PrintStack()
		return kafkaUrl, err
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		log.Println("DEBUG: remove container kafka", c.Terminate(context.Background()))
	}()

	err = Dockerlog(ctx, c, "KAFKA")
	if err != nil {
		return kafkaUrl, err
	}

	containerPort, err := c.MappedPort(ctx, "9092/tcp")
	if err != nil {
		debug.PrintStack()
		return kafkaUrl, err
	}
	err = Forward(ctx, kafkaport, hostIp+":"+containerPort.Port())
	if err != nil {
		debug.PrintStack()
		return kafkaUrl, err
	}

	err = retry(1*time.Minute, func() error {
		return tryKafkaConn(kafkaUrl)
	})
	if err != nil {
		debug.PrintStack()
		return kafkaUrl, err
	}

	return kafkaUrl, err
}

func tryKafkaConn(kafkaUrl string) error {
	log.Println("try kafka connection to " + kafkaUrl + "...")
	conn, err := kafka.Dial("tcp", kafkaUrl)
	if err != nil {
		log.Println(err)
		return err
	}
	defer conn.Close()
	brokers, err := conn.Brokers()
	if err != nil {
		log.Println(err)
		return err
	}
	if len(brokers) == 0 {
		err = errors.New("missing brokers")
		log.Println(err)
		return err
	}
	log.Println("kafka connection ok")
	return nil
}

func Zookeeper(ctx context.Context, wg *sync.WaitGroup) (hostPort string, ipAddress string, err error) {
	log.Println("start zookeeper")
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "wurstmeister/zookeeper:latest",
			Tmpfs: map[string]string{"/opt/zookeeper-3.4.13/data": "rw"},
			WaitingFor: wait.ForAll(
				wait.ForLog("binding to port"),
				wait.ForListeningPort("2181/tcp"),
				wait.ForNop(waitretry(1*time.Minute, func(ctx context.Context, target wait.StrategyTarget) error {
					log.Println("try zk connection...")
					zookeeper := kazoo.NewConfig()
					host, err := target.Host(ctx)
					if err != nil {
						log.Println("host", err)
						return err
					}
					port, err := target.MappedPort(ctx, "2181/tcp")
					if err != nil {
						log.Println("port", err)
						return err
					}
					zk, chroot := kazoo.ParseConnectionString(host + ":" + port.Port())
					zookeeper.Chroot = chroot
					kz, err := kazoo.NewKazoo(zk, zookeeper)
					if err != nil {
						log.Println("kazoo", err)
						return err
					}
					_, err = kz.Brokers()
					if err != nil && strings.TrimSpace(err.Error()) != strings.TrimSpace("zk: node does not exist") {
						log.Println("brokers", err)
						return err
					}
					return nil
				}))),
			ExposedPorts:    []string{"2181/tcp"},
			AlwaysPullImage: true,
		},
		Started: true,
	})
	if err != nil {
		return "", "", err
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		log.Println("DEBUG: remove container zookeeper", c.Terminate(context.Background()))
	}()

	ipAddress, err = c.ContainerIP(ctx)
	if err != nil {
		return "", "", err
	}
	temp, err := c.MappedPort(ctx, "2181/tcp")
	if err != nil {
		return "", "", err
	}
	hostPort = temp.Port()

	return hostPort, ipAddress, err
}

func waitretry(timeout time.Duration, f func(ctx context.Context, target wait.StrategyTarget) error) func(ctx context.Context, target wait.StrategyTarget) error {
	return func(ctx context.Context, target wait.StrategyTarget) (err error) {
		return retry(timeout, func() error {
			return f(ctx, target)
		})
	}
}

func retry(timeout time.Duration, f func() error) (err error) {
	err = errors.New("initial")
	start := time.Now()
	for i := int64(1); err != nil && time.Since(start) < timeout; i++ {
		err = f()
		if err != nil {
			log.Println("ERROR: :", err)
			wait := time.Duration(i) * time.Second
			if time.Since(start)+wait < timeout {
				log.Println("ERROR: retry after:", wait.String())
				time.Sleep(wait)
			} else {
				time.Sleep(time.Since(start) + wait - timeout)
				return f()
			}
		}
	}
	return err
}

func Dockerlog(ctx context.Context, container testcontainers.Container, name string) error {
	l, err := container.Logs(ctx)
	if err != nil {
		return err
	}
	out := &LogWriter{logger: log.New(os.Stdout, "["+name+"] ", log.LstdFlags)}
	go func() {
		_, err := io.Copy(out, l)
		if err != nil {
			log.Println("ERROR: unable to copy docker log", err)
		}
	}()
	return nil
}

type LogWriter struct {
	logger *log.Logger
}

func (this *LogWriter) Write(p []byte) (n int, err error) {
	this.logger.Print(string(p))
	return len(p), nil
}

func Forward(ctx context.Context, fromPort int, toAddr string) error {
	log.Println("forward", fromPort, "to", toAddr)
	incoming, err := net.Listen("tcp", fmt.Sprintf(":%d", fromPort))
	if err != nil {
		return err
	}
	go func() {
		defer log.Println("closed forward incoming")
		<-ctx.Done()
		incoming.Close()
	}()
	go func() {
		for {
			client, err := incoming.Accept()
			if err != nil {
				log.Println("FORWARD ERROR:", err)
				return
			}
			go handleForwardClient(client, toAddr)
		}
	}()
	return nil
}

func handleForwardClient(client net.Conn, addr string) {
	log.Println("new forward client")
	target, err := net.Dial("tcp", addr)
	if err != nil {
		log.Println("FORWARD ERROR:", err)
		return
	}
	go func() {
		defer target.Close()
		defer client.Close()
		io.Copy(target, client)
	}()
	go func() {
		defer target.Close()
		defer client.Close()
		io.Copy(client, target)
	}()
}

func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func createTestAspects(ctx context.Context, config configuration.Config, ids ...string) func(t *testing.T) {
	return func(t *testing.T) {
		p, err := k.NewProducer(ctx, config.KafkaUrl, "aspects", true)
		if err != nil {
			t.Error(err)
			return
		}
		for _, id := range ids {
			t.Run("create "+id, createTestAspect(p, id))
		}
	}
}

func createTestAspect(p *k.Producer, id string) func(t *testing.T) {
	return func(t *testing.T) {
		aspectMsg, aspectCmd, err := getAspectTestObj(id, map[string]interface{}{
			"name":     id + "_name",
			"rdf_type": "aspect_type",
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

func testRequest(config configuration.Config, method string, path string, body interface{}, expectedStatusCode int, expected interface{}) func(t *testing.T) {
	return testRequestWithToken(config, testtoken, method, path, body, expectedStatusCode, expected)
}

func testRequestWithToken(config configuration.Config, token string, method string, path string, body interface{}, expectedStatusCode int, expected interface{}) func(t *testing.T) {
	return func(t *testing.T) {
		var requestBody io.Reader
		if body != nil {
			temp := new(bytes.Buffer)
			err := json.NewEncoder(temp).Encode(body)
			if err != nil {
				t.Error(err)
				return
			}
			requestBody = temp
		}

		req, err := http.NewRequest(method, "http://localhost:"+config.ServerPort+path, requestBody)
		if err != nil {
			t.Error(err)
			return
		}
		req.Header.Set("Authorization", token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
			return
		}
		if resp.StatusCode != expectedStatusCode {
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
			return
		}

		if expected != nil {
			temp, err := json.Marshal(expected)
			if err != nil {
				t.Error(err)
				return
			}
			var normalizedExpected interface{}
			err = json.Unmarshal(temp, &normalizedExpected)
			if err != nil {
				t.Error(err)
				return
			}

			var actual interface{}
			err = json.NewDecoder(resp.Body).Decode(&actual)
			if err != nil {
				t.Error(err)
				return
			}

			if !reflect.DeepEqual(actual, normalizedExpected) {
				a, _ := json.Marshal(actual)
				e, _ := json.Marshal(normalizedExpected)
				t.Error("\n", string(a), "\n", string(e))
				return
			}
		}
	}
}

func getTestAspectResult(id string) map[string]interface{} {
	return getTestAspectResultWithPermissionHolders(id, []string{"testOwner"}, false)
}

func getTestAspectResultWithPermissionHolders(id string, userList []string, shared bool) map[string]interface{} {
	//map[creator:testOwner id:aaspect name:aaspect_name permissions:map[a:true r:true w:true x:true] shared:false
	sort.Strings(userList)
	return map[string]interface{}{
		"creator": "testOwner",
		"id":      id,
		"name":    id + "_name",
		"permissions": map[string]bool{
			"a": true,
			"r": true,
			"w": true,
			"x": true,
		},
		"raw": map[string]interface{}{
			"name":     id + "_name",
			"rdf_type": "aspect_type",
		},
		"permission_holders": map[string][]string{
			"admin_users":   userList,
			"execute_users": userList,
			"read_users":    userList,
			"write_users":   userList,
		},
		"shared": shared,
	}
}

func saveTestDevice(w *worker.Worker, resource string, id string, fields map[string]interface{}) func(t *testing.T) {
	return func(t *testing.T) {
		msg, cmd, err := getDeviceTestObj(id, fields)
		if err != nil {
			t.Error(err)
			return
		}
		err = w.UpdateFeatures(resource, msg, cmd)
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func getDeviceTestObj(id string, obj interface{}) (msg []byte, command model.CommandWrapper, err error) {
	text := `{
		"command": "PUT",
		"id": "%s",
		"owner": "testOwner",
		"device": %s
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

func createTestToken(user string, groups []string) auth.Token {
	return auth.Token{
		Token:       "",
		Sub:         user,
		RealmAccess: map[string][]string{"roles": groups},
	}
}

func getAspectTestObj(id string, obj map[string]interface{}) (msg []byte, command model.CommandWrapper, err error) {
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
