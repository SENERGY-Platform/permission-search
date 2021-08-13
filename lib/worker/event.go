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
	"log"
	"time"
)

func InitEventHandling(ctx context.Context, config configuration.Config, worker *Worker) (err error) {
	err = kafka.NewConsumer(ctx, config.KafkaUrl, config.GroupId, config.PermTopic, func(msg []byte) error {
		return worker.HandlePermissionCommand(msg)
	})
	if err != nil {
		return err
	}

	err = kafka.NewConsumer(ctx, config.KafkaUrl, config.GroupId, config.UserTopic, func(msg []byte) error {
		return worker.HandleUserCommand(msg)
	})
	if err != nil {
		return err
	}

	err = kafka.NewConsumer(ctx, config.KafkaUrl, config.GroupId, config.UserTopic, func(msg []byte) error {
		return worker.HandleUserCommand(msg)
	})
	if err != nil {
		return err
	}

	log.Println("init features handlers", config.ResourceList)
	for _, resource := range config.ResourceList {
		log.Println("init handler for", resource)
		f := worker.GetResourceCommandHandler(resource)
		err = kafka.NewConsumer(ctx, config.KafkaUrl, config.GroupId, resource, func(msg []byte) error {
			return f(msg)
		})
		if err != nil {
			return err
		}
	}

	log.Println("init annotation handlers", config.AnnotationResourceIndex)
	for topic, resources := range config.AnnotationResourceIndex {
		log.Println("init annotation handler for", topic)
		f := worker.GetAnnotationHandler(topic, resources)
		err = kafka.NewConsumer(ctx, config.KafkaUrl, config.GroupId+"_annotation", topic, func(msg []byte) error {
			return f(msg)
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

func (this *Worker) HandleUserCommand(msg []byte) (err error) {
	log.Println(this.config.UserTopic, string(msg))
	command := model.UserCommandMsg{}
	err = json.Unmarshal(msg, &command)
	if err != nil {
		return
	}
	switch command.Command {
	case "DELETE":
		if command.Id != "" {
			return this.DeleteUser(command.Id)
		}
	}
	log.Println("WARNING: unable to handle user command: " + string(msg))
	return nil
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
	ctx, _ := context.WithTimeout(context.Background(), 20*time.Second)
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
	if exists {
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

		_, err = this.query.GetClient().Index().Index(resource).Id(idStr).IfPrimaryTerm(version.PrimaryTerm).IfSeqNo(version.SeqNo).BodyJson(entry).Do(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}
