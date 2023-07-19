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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/worker/kafka"
	"github.com/opensearch-project/opensearch-go/opensearchutil"
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

	resourceTopics := config.ResourceList

	annotationTopics := []string{}
	for topic, _ := range config.AnnotationResourceIndex {
		annotationTopics = append(annotationTopics, topic)
	}

	topicFilter := replacePlaceholders(config.TopicFilter, map[string][]string{
		"{{.annotations}}": annotationTopics,
		"{{.resources}}":   resourceTopics,
	})

	resourceTopics = filterSpecialisation(resourceTopics, topicFilter)
	annotationTopics = filterSpecialisation(annotationTopics, topicFilter)

	log.Println("init features handlers", resourceTopics)
	handlers := map[string]func(delivery []byte) error{}
	for _, resource := range resourceTopics {
		log.Println("init handler for", resource)
		handlers[resource] = worker.GetResourceCommandHandler(resource)
	}

	err = kafka.NewConsumerWithMultipleTopics(ctx, config.KafkaUrl, config.GroupId, resourceTopics, config.Debug, func(topic string, msg []byte) error {
		f, ok := handlers[topic]
		if !ok {
			log.Println("ERROR: unknown topic handler ", topic)
			return nil
		}
		return f(msg)
	}, func(topic string, err error) {
		config.HandleFatalError(err)
	})

	log.Println("init annotation handlers", annotationTopics)
	annotationHandlers := map[string]func(delivery []byte) error{}

	for _, topic := range annotationTopics {
		log.Println("init annotation handler for", topic)
		annotationHandlers[topic] = worker.GetAnnotationHandler(topic, config.AnnotationResourceIndex[topic])
	}

	err = kafka.NewConsumerWithMultipleTopics(ctx, config.KafkaUrl, config.GroupId+"_annotation", annotationTopics, config.Debug, func(topic string, msg []byte) error {
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
	return
}

func replacePlaceholders(list []string, replace map[string][]string) (result []string) {
	for _, element := range list {
		if repl, ok := replace[element]; ok {
			result = append(result, repl...)
		} else {
			result = append(result, element)
		}
	}
	return result
}

func filterSpecialisation(list []string, in []string) (result []string) {
	if len(in) == 0 {
		return list
	}
	index := map[string]bool{}
	for _, topic := range in {
		index[topic] = true
	}
	for _, topic := range list {
		if index[topic] {
			result = append(result, topic)
		}
	}
	return result
}

func (this *Worker) HandlePermissionCommand(msg []byte) (err error) {
	log.Println(this.config.PermTopic, string(msg))
	command := model.PermCommandMsg{}
	err = json.Unmarshal(msg, &command)
	if err != nil {
		return
	}
	defer func() {
		if err == nil {
			err = this.SendDone(model.Done{
				ResourceKind: command.Kind,
				ResourceId:   command.Resource,
				Command:      "RIGHTS",
			})
		}
	}()
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
		defer func() {
			if err == nil {
				err = this.SendDone(model.Done{
					ResourceKind: resourceName,
					ResourceId:   command.Id,
					Command:      command.Command,
				})
			}
		}()
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

	exists, err := this.query.ResourceExists(resource, idStr)
	if err != nil {
		return err
	}
	if this.config.Debug && !exists {
		log.Println("WARNING: _id is unknown in", resource, idStr)
	}

	if this.config.UseBulkWorkerForAnnotations {
		annotations := map[string]interface{}{}
		for fieldName, fieldValue := range fields {
			if fieldName != "_id" {
				annotations[fieldName] = fieldValue
			}
		}
		buffer, err := json.Marshal(map[string]interface{}{"doc": map[string]interface{}{"annotations": annotations}})
		if err != nil {
			return err
		}
		err = this.bulk.Add(context.Background(),
			opensearchutil.BulkIndexerItem{
				Action:     "update",
				Index:      resource,
				DocumentID: idStr,

				Body: bytes.NewReader(buffer),
			})
		if err != nil {
			return err
		}
	} else {
		entry, version, err := this.query.GetResourceEntry(resource, idStr)
		if err != nil {
			return err
		}
		if entry.Annotations == nil {
			entry.Annotations = map[string]interface{}{}
		}
		for fieldName, fieldValue := range fields {
			if fieldName != "_id" {
				entry.Annotations[fieldName] = fieldValue
			}
		}
		ctx, _ := context.WithTimeout(context.Background(), 20*time.Second)
		client := this.query.GetClient()
		resp, err := client.Index(
			resource,
			opensearchutil.NewJSONReader(entry),
			client.Index.WithDocumentID(idStr),
			client.Index.WithIfPrimaryTerm(int(version.PrimaryTerm)),
			client.Index.WithIfSeqNo(int(version.SeqNo)),
			client.Index.WithContext(ctx),
		)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.IsError() {
			return errors.New(resp.String())
		}
	}

	return nil
}
