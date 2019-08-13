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
	"encoding/json"
	"errors"
	"log"
)


func InitEventHandling() (err error) {

	_, err = NewConsumer(Config.ZookeeperUrl, Config.GroupId, Config.PermTopic, func(topic string, msg []byte) error {
		return handlePermissionCommand(msg)
	}, func(err error, consumer *Consumer) {
		log.Fatal(err)
	})
	if err != nil {
		log.Fatal("ERROR: while initializing kafka connection", err)
		return
	}

	_, err = NewConsumer(Config.ZookeeperUrl, Config.GroupId, Config.UserTopic, func(topic string, msg []byte) error {
		return handleUserCommand(msg)
	}, func(err error, consumer *Consumer) {
		log.Fatal(err)
	})
	if err != nil {
		log.Fatal("ERROR: while initializing kafka connection", err)
		return
	}

	_, err = NewConsumer(Config.ZookeeperUrl, Config.GroupId, Config.UserTopic, func(topic string, msg []byte) error {
		return handleUserCommand(msg)
	}, func(err error, consumer *Consumer) {
		log.Fatal(err)
	})
	if err != nil {
		log.Fatal("ERROR: while initializing kafka connection", err)
		return
	}

	log.Println("init features handler", Config.ResourceList)
	for _, resource := range Config.ResourceList {
		f := getResourceCommandHandler(resource)
		_, err = NewConsumer(Config.ZookeeperUrl, Config.GroupId, resource, func(topic string, msg []byte) error {
			return f(msg)
		}, func(err error, consumer *Consumer) {
			log.Fatal(err)
		})
		if err != nil {
			log.Fatal("ERROR: while initializing kafka connection", err)
			return
		}
	}

	return
}

func handlePermissionCommand(msg []byte) (err error) {
	log.Println(Config.PermTopic, string(msg))
	command := PermCommandMsg{}
	err = json.Unmarshal(msg, &command)
	if err != nil {
		return
	}
	switch command.Command {
	case "PUT":
		if command.User != "" {
			return SetUserRight(command.Kind, command.Resource, command.User, command.Right)
		}
		if command.Group != "" {
			return SetGroupRight(command.Kind, command.Resource, command.Group, command.Right)
		}
	case "DELETE":
		if command.User != "" {
			return DeleteUserRight(command.Kind, command.Resource, command.User)
		}
		if command.Group != "" {
			return DeleteGroupRight(command.Kind, command.Resource, command.Group)
		}
	}
	return errors.New("unable to handle permission command: " + string(msg))
}

func handleUserCommand(msg []byte) (err error) {
	log.Println(Config.UserTopic, string(msg))
	command := UserCommandMsg{}
	err = json.Unmarshal(msg, &command)
	if err != nil {
		return
	}
	switch command.Command {
	case "DELETE":
		if command.Id != "" {
			return DeleteUser(command.Id)
		}
	}
	log.Println("WARNING: unable to handle user command: " + string(msg))
	return nil
}

func getResourceCommandHandler(resourceName string) func(delivery []byte) error {
	return func(msg []byte) (err error) {
		command := CommandWrapper{}
		err = json.Unmarshal(msg, &command)
		if err != nil {
			return
		}
		switch command.Command {
		case "PUT":
			return UpdateFeatures(resourceName, msg, command)
		case "DELETE":
			return DeleteFeatures(resourceName, command)
		}
		return errors.New("unable to handle command: " + resourceName + " " + string(msg))
	}
}
