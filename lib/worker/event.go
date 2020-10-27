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
)

func InitEventHandling(ctx context.Context, config configuration.Config, worker *Worker) (err error) {
	err = kafka.NewConsumer(ctx, config.ZookeeperUrl, config.GroupId, config.PermTopic, func(msg []byte) error {
		return worker.HandlePermissionCommand(msg)
	})
	if err != nil {
		return err
	}

	err = kafka.NewConsumer(ctx, config.ZookeeperUrl, config.GroupId, config.UserTopic, func(msg []byte) error {
		return worker.HandleUserCommand(msg)
	})
	if err != nil {
		return err
	}

	err = kafka.NewConsumer(ctx, config.ZookeeperUrl, config.GroupId, config.UserTopic, func(msg []byte) error {
		return worker.HandleUserCommand(msg)
	})
	if err != nil {
		return err
	}

	log.Println("init features handler", config.ResourceList)
	for _, resource := range config.ResourceList {
		log.Println("init handler for", resource)
		f := worker.GetResourceCommandHandler(resource)
		err = kafka.NewConsumer(ctx, config.ZookeeperUrl, config.GroupId, resource, func(msg []byte) error {
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
