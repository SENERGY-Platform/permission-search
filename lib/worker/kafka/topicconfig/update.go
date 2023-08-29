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

package topicconfig

import (
	"errors"
	"github.com/IBM/sarama"
	"github.com/segmentio/kafka-go"
	"log"
	"net"
	"strconv"
)

func Ensure(bootstrapUrl string, topic string, config map[string]string) (err error) {
	controller, err := getKafkaController(bootstrapUrl)
	if err != nil {
		log.Println("ERROR: unable to find controller", err)
		return err
	}
	if controller == "" {
		log.Println("ERROR: unable to find controller")
		return errors.New("unable to find controller")
	}
	return EnsureWithBroker(controller, topic, config)
}

func EnsureWithBroker(broker string, topic string, config map[string]string) (err error) {
	sconfig := sarama.NewConfig()
	sconfig.Version = sarama.V2_4_0_0
	admin, err := sarama.NewClusterAdmin([]string{broker}, sconfig)
	if err != nil {
		return err
	}

	temp := map[string]*string{}
	for key, value := range config {
		tempValue := value
		temp[key] = &tempValue
	}

	err = set(admin, topic, temp)
	if err != nil {
		log.Println("WARNING: ", err)
		log.Println("create topic: ", topic, config)
		err = create(admin, topic, temp)
	}

	return err
}

func set(admin sarama.ClusterAdmin, topic string, config map[string]*string) (err error) {
	return admin.AlterConfig(sarama.TopicResource, topic, config, false)
}

func create(admin sarama.ClusterAdmin, topic string, config map[string]*string) (err error) {
	return admin.CreateTopic(topic, &sarama.TopicDetail{
		NumPartitions:     1,
		ReplicationFactor: 1,
		ConfigEntries:     config,
	}, false)
}

func getKafkaController(bootstrapUrl string) (result string, err error) {
	conn, err := kafka.Dial("tcp", bootstrapUrl)
	if err != nil {
		return result, err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return result, err
	}
	return net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)), nil
}
