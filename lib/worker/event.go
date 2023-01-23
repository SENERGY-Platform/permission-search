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

package worker

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/worker/kafka"
	"github.com/olivere/elastic/v7"
	"log"
	"time"
)

const permissionsCommandErrorMsg = `ERROR: unable to handle permissions command
	--> ignore message and commit to kafka to ensure continuing consumption
	`

func InitEventHandling(ctx context.Context, config configuration.Config, worker *Worker) (err error) {
	err = kafka.NewConsumer(ctx, config.KafkaUrl, config.GroupId, config.PermTopic, func(msg []byte) error {
		err := worker.HandlePermissionCommand(msg)
		if err != nil {
			log.Println(permissionsCommandErrorMsg, err)
		}
		return nil
	}, func(err error) {
		config.HandleFatalError(err)
	})
	if err != nil {
		return err
	}

	log.Println("init features handlers", config.ResourceList)
	handlers := map[string]func(delivery []byte) error{}
	for _, resource := range config.ResourceList {
		log.Println("init handler for", resource)
		handlers[resource] = worker.GetResourceCommandHandler(resource)
	}
	err = kafka.NewConsumerWithMultipleTopics(ctx, config.KafkaUrl, config.GroupId, config.ResourceList, 1*time.Second, func(topic string, msg []byte) error {
		f, ok := handlers[topic]
		if !ok {
			log.Println("ERROR: unknown topic handler ", topic)
			return nil
		}
		return f(msg)
	}, func(topic string, err error) {
		config.HandleFatalError(err)
	})

	log.Println("init annotation handlers", config.AnnotationResourceIndex)
	annotationHandlers := map[string]func(delivery []byte) error{}
	annotationTopics := []string{}
	for topic, resources := range config.AnnotationResourceIndex {
		log.Println("init annotation handler for", topic)
		annotationHandlers[topic] = worker.GetAnnotationHandler(topic, resources)
		annotationTopics = append(annotationTopics, topic)
	}
	err = kafka.NewConsumerWithMultipleTopics(ctx, config.KafkaUrl, config.GroupId+"_annotation", annotationTopics, 10*time.Second, func(topic string, msg []byte) error {
		f, ok := annotationHandlers[topic]
		if !ok {
			log.Println("ERROR: unknown annotation topic handler ", topic)
			return nil
		}
		return f(msg)
	}, func(topic string, err error) {
		config.HandleFatalError(err)
	})
	if err != nil {
		return err
	}

	for topic, resources := range config.AnnotationResourceIndex {
		log.Println("init annotation handler for", topic)
		f := worker.GetAnnotationHandler(topic, resources)
		err = kafka.NewConsumer(ctx, config.KafkaUrl, config.GroupId+"_annotation", topic, func(msg []byte) error {
			return f(msg)
		}, func(err error) {
			config.HandleFatalError(err)
		})
		if err != nil {
			return err
		}
	}
	return
}

func (this *Worker) HandlePermissionCommand(msg []byte) (err error) {
	log.Println(this.config.PermTopic, string(msg))
	command := model.PermCommandMsg{}
	err = json.Unmarshal(msg, &command)
	if err != nil {
		return
	}
	switch command.Command {
	case "PUT":
		if command.User != "" {
			return this.SetUserRight(command.Kind, command.Resource, command.User, command.Right)
		}
		if command.Group != "" {
			return this.SetGroupRight(command.Kind, command.Resource, command.Group, command.Right)
		}
	case "DELETE":
		if command.User != "" {
			return this.DeleteUserRight(command.Kind, command.Resource, command.User)
		}
		if command.Group != "" {
			return this.DeleteGroupRight(command.Kind, command.Resource, command.Group)
		}
	}
	return errors.New("unable to handle permission command: " + string(msg))
}

func (this *Worker) GetResourceCommandHandler(resourceName string) func(delivery []byte) error {
	return func(msg []byte) (err error) {
		if this.config.Debug {
			log.Println("receive command", resourceName, string(msg))
		}
		command := model.CommandWrapper{}
		err = json.Unmarshal(msg, &command)
		if err != nil {
			return
		}
		switch command.Command {
		case "RIGHTS":
			//RIGHTS commands are expected to be produced in the same partition as PUT and DELETE commands for the same resource/id
			//but the kafka.key has to be different from the keys used by PUT/DELETE commands, to ensure separate compaction between PUT/DELETE commands
			//this can be achieved by using a custom Balancer with the kafka producer. an example can be seen with kafka.KeySeparationBalancer and NewProducerWithKeySeparationBalancer()
			//if the NewProducerWithKeySeparationBalancer() is used a PUT key would be for example 'my-device-id' and the RIGHTS key would be 'my-device-id/rights'
			return this.UpdateRights(resourceName, msg, command)
		case "PUT":
			return this.UpdateFeatures(resourceName, msg, command)
		case "DELETE":
			return this.DeleteFeatures(resourceName, command)
		}
		return errors.New("unable to handle command: " + resourceName + " " + string(msg))
	}
}

func (this *Worker) GetAnnotationHandler(annotationTopic string, resources []string) func(delivery []byte) error {
	return func(msg []byte) error {
		if this.config.Debug {
			log.Println("receive annotation", annotationTopic, string(msg))
		}
		for _, resource := range resources {
			err := this.HandleAnnotationMsg(annotationTopic, resource, msg)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func (this *Worker) HandleAnnotationMsg(annotationTopic string, resource string, msg []byte) error {
	fields, err := this.MsgToAnnotations(resource, annotationTopic, msg)
	if err != nil {
		return err
	}
	id, ok := fields["_id"]
	if !ok {
		log.Println("WARNING: missing _id field in annotation --> ignore", resource, annotationTopic)
		return nil
	}
	idStr, ok := id.(string)
	if !ok {
		log.Println("WARNING: _id field in annotation is not string --> ignore ", resource, annotationTopic)
		return nil
	}

	annotations := map[string]interface{}{}
	for fieldName, fieldValue := range fields {
		if fieldName != "_id" {
			annotations[fieldName] = fieldValue
		}
	}

	this.bulk.Add(elastic.NewBulkUpdateRequest().Index(resource).Id(idStr).Doc(map[string]interface{}{"annotations": annotations}))
	return nil
}
