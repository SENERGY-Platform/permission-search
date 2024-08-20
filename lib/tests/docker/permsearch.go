/*
 * Copyright 2021 InfAI (CC SES)
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

package docker

import (
	"context"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"log"
	"sync"
)

func PermissionSearch(ctx context.Context, wg *sync.WaitGroup, containerLog bool, kafkaUrl string, dbIp string) (hostPort string, ipAddress string, err error) {
	log.Println("start PermissionSearch")
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "ghcr.io/senergy-platform/permission-search:dev",
			Env: map[string]string{
				"KAFKA_URL":                        kafkaUrl,
				"OPEN_SEARCH_URLS":                 "https://" + dbIp + ":9200",
				"OPEN_SEARCH_USERNAME":             "admin",
				"OPEN_SEARCH_PASSWORD":             OpenSearchTestPw,
				"OPEN_SEARCH_INSECURE_SKIP_VERIFY": "true",
			},
			ExposedPorts:    []string{"8080/tcp"},
			WaitingFor:      wait.ForListeningPort("8080/tcp"),
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
		log.Println("DEBUG: remove container PermissionSearch", c.Terminate(context.Background()))
	}()

	if containerLog {
		err = Dockerlog(ctx, c, "PERMISSIONS-SEARCH")
		if err != nil {
			return "", "", err
		}
	}

	ipAddress, err = c.ContainerIP(ctx)
	if err != nil {
		return "", "", err
	}
	temp, err := c.MappedPort(ctx, "8080/tcp")
	if err != nil {
		return "", "", err
	}
	hostPort = temp.Port()

	return hostPort, ipAddress, err
}
