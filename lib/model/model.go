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

package model

import (
	"encoding/json"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"log"
)

func (entry *Entry) SetDefaultPermissions(config configuration.Config, kind string, owner string) {
	if owner != "" {
		entry.AdminUsers = []string{owner}
		entry.ReadUsers = []string{owner}
		entry.WriteUsers = []string{owner}
		entry.ExecuteUsers = []string{owner}
	}
	for group, rights := range config.Resources[kind].InitialGroupRights {
		entry.AddGroupRights(group, rights)
	}
	return
}

func (entry *Entry) AddUserRights(user string, rights string) {
	for _, right := range rights {
		switch right {
		case 'a':
			entry.AdminUsers = append(entry.AdminUsers, user)
		case 'r':
			entry.ReadUsers = append(entry.ReadUsers, user)
		case 'w':
			entry.WriteUsers = append(entry.WriteUsers, user)
		case 'x':
			entry.ExecuteUsers = append(entry.ExecuteUsers, user)
		}
	}
}

func (entry *Entry) RemoveUserRights(user string) {
	entry.AdminUsers = listRemove(entry.AdminUsers, user)
	entry.ReadUsers = listRemove(entry.ReadUsers, user)
	entry.WriteUsers = listRemove(entry.WriteUsers, user)
	entry.ExecuteUsers = listRemove(entry.ExecuteUsers, user)
}

func (entry *Entry) AddGroupRights(group string, rights string) {
	for _, right := range rights {
		switch right {
		case 'a':
			entry.AdminGroups = append(entry.AdminGroups, group)
		case 'r':
			entry.ReadGroups = append(entry.ReadGroups, group)
		case 'w':
			entry.WriteGroups = append(entry.WriteGroups, group)
		case 'x':
			entry.ExecuteGroups = append(entry.AdminGroups, group)
		}
	}
}

func (entry *Entry) RemoveGroupRights(group string) {
	entry.AdminGroups = listRemove(entry.AdminGroups, group)
	entry.ReadGroups = listRemove(entry.ReadGroups, group)
	entry.WriteGroups = listRemove(entry.WriteGroups, group)
	entry.ExecuteGroups = listRemove(entry.ExecuteGroups, group)
}

func listRemove(list []string, element string) (result []string) {
	for _, e := range list {
		if e != element {
			result = append(result, e)
		}
	}
	return
}

type PermCommandMsg struct {
	Command  string `json:"command"`
	Kind     string
	Resource string
	User     string
	Group    string
	Right    string
}

type UserCommandMsg struct {
	Command string `json:"command"`
	Id      string `json:"id"`
}

type CommandWrapper struct {
	Command              string `json:"command"`
	Id                   string `json:"id"`
	Owner                string `json:"owner"`
	StrictWaitBeforeDone bool   `json:"strict_wait_before_done"`
}

//RIGHTS commands are expected to be produced in the same partition as PUT and DELETE commands for the same resource/id
//but the kafka.key has to be different from the keys used by PUT/DELETE commands, to ensure separate compaction between PUT/DELETE commands
//this can be achieved by using a custom Balancer with the kafka producer. an example can be seen with kafka.KeySeparationBalancer and NewProducerWithKeySeparationBalancer()
//if the NewProducerWithKeySeparationBalancer() is used a PUT key would be for example 'my-device-id' and the RIGHTS key would be 'my-device-id/rights'

type CommandWithRights struct {
	Command string              `json:"command"`
	Id      string              `json:"id"`
	Rights  *ResourceRightsBase `json:"rights"`
}

type ResourceRightsBase struct {
	UserRights  map[string]Right `json:"user_rights"`
	GroupRights map[string]Right `json:"group_rights"`
}

type ResourceRights struct {
	ResourceRightsBase
	ResourceId string                 `json:"resource_id"`
	Features   map[string]interface{} `json:"features"`
	Creator    string                 `json:"creator"`
}

type Right struct {
	Read         bool `json:"read"`
	Write        bool `json:"write"`
	Execute      bool `json:"execute"`
	Administrate bool `json:"administrate"`
}

type Entry struct {
	Resource      string                 `json:"resource"`
	Features      map[string]interface{} `json:"features"`
	Annotations   map[string]interface{} `json:"annotations"`
	AdminUsers    []string               `json:"admin_users"`
	AdminGroups   []string               `json:"admin_groups"`
	ReadUsers     []string               `json:"read_users"`
	ReadGroups    []string               `json:"read_groups"`
	WriteUsers    []string               `json:"write_users"`
	WriteGroups   []string               `json:"write_groups"`
	ExecuteUsers  []string               `json:"execute_users"`
	ExecuteGroups []string               `json:"execute_groups"`
	Creator       string                 `json:"creator"`
}

// EntryResult is ment to be used in combination with a resource model
// is intended to be used with client.Query()
// example:
//
//	type Device struct{
//		Id   string `json:"id"`
//		Name string `json:"name"`
//	}
//
//	type DeviceResult struct {
//		EntryResult
//		Device
//	}
//
//	func main() {
//		client.Query[DeviceResult](c, token, model.QueryMessage{Resource:"devices"})
//		client.Query[Device](c, token, model.QueryMessage{Resource:"devices"})
//	}
type EntryResult struct {
	Creator           string                       `json:"creator"`
	Annotations       map[string]interface{}       `json:"annotations"`
	Permissions       map[string]bool              `json:"permissions"`
	PermissionHolders EntryResultPermissionHolders `json:"permission_holders"`
	Shared            bool                         `json:"shared"`
}

type EntryResultPermissionHolders struct {
	AdminUsers   []string `json:"admin_users"`
	ReadUsers    []string `json:"read_users"`
	WriteUsers   []string `json:"write_users"`
	ExecuteUsers []string `json:"execute_users"`
}

func (this *Entry) SetResourceRights(rights ResourceRightsBase) {
	for group, right := range rights.GroupRights {
		if right.Administrate {
			this.AdminGroups = append(this.AdminGroups, group)
		}
		if right.Execute {
			this.ExecuteGroups = append(this.ExecuteGroups, group)
		}
		if right.Write {
			this.WriteGroups = append(this.WriteGroups, group)
		}
		if right.Read {
			this.ReadGroups = append(this.ReadGroups, group)
		}
	}
	for user, right := range rights.UserRights {
		if right.Administrate {
			this.AdminUsers = append(this.AdminUsers, user)
		}
		if right.Execute {
			this.ExecuteUsers = append(this.ExecuteUsers, user)
		}
		if right.Write {
			this.WriteUsers = append(this.WriteUsers, user)
		}
		if right.Read {
			this.ReadUsers = append(this.ReadUsers, user)
		}
	}
}

func (entry Entry) ToResourceRights() (result ResourceRights) {
	result.ResourceId = entry.Resource
	result.Features = entry.Features
	result.Creator = entry.Creator
	result.UserRights = map[string]Right{}
	for _, user := range entry.AdminUsers {
		if _, ok := result.UserRights[user]; !ok {
			result.UserRights[user] = Right{}
		}
		right := result.UserRights[user]
		right.Administrate = true
		result.UserRights[user] = right
	}
	for _, user := range entry.ReadUsers {
		if _, ok := result.UserRights[user]; !ok {
			result.UserRights[user] = Right{}
		}
		right := result.UserRights[user]
		right.Read = true
		result.UserRights[user] = right
	}
	for _, user := range entry.WriteUsers {
		if _, ok := result.UserRights[user]; !ok {
			result.UserRights[user] = Right{}
		}
		right := result.UserRights[user]
		right.Write = true
		result.UserRights[user] = right
	}
	for _, user := range entry.ExecuteUsers {
		if _, ok := result.UserRights[user]; !ok {
			result.UserRights[user] = Right{}
		}
		right := result.UserRights[user]
		right.Execute = true
		result.UserRights[user] = right
	}

	result.GroupRights = map[string]Right{}
	for _, group := range entry.AdminGroups {
		if _, ok := result.GroupRights[group]; !ok {
			result.GroupRights[group] = Right{}
		}
		right := result.GroupRights[group]
		right.Administrate = true
		result.GroupRights[group] = right
	}
	for _, group := range entry.ReadGroups {
		if _, ok := result.GroupRights[group]; !ok {
			result.GroupRights[group] = Right{}
		}
		right := result.GroupRights[group]
		right.Read = true
		result.GroupRights[group] = right
	}
	for _, group := range entry.WriteGroups {
		if _, ok := result.GroupRights[group]; !ok {
			result.GroupRights[group] = Right{}
		}
		right := result.GroupRights[group]
		right.Write = true
		result.GroupRights[group] = right
	}
	for _, group := range entry.ExecuteGroups {
		if _, ok := result.GroupRights[group]; !ok {
			result.GroupRights[group] = Right{}
		}
		right := result.GroupRights[group]
		right.Execute = true
		result.GroupRights[group] = right
	}
	return
}

const PermissionMapping = `{
	"admin_groups":   {"type": "keyword"},
	"admin_users":    {"type": "keyword"},
	"execute_groups": {"type": "keyword"},
	"execute_users":  {"type": "keyword"},
	"read_groups":    {"type": "keyword"},
	"read_users":     {"type": "keyword"},
	"resource":       {"type": "keyword"},
	"write_groups":   {"type": "keyword"},
	"write_users":    {"type": "keyword"},
	"creator":    	  {"type": "keyword"},
	"feature_search": {"type": "text", "analyzer": "custom_analyzer", "search_analyzer": "custom_search_analyzer"}
}`

func CreateMapping(config configuration.Config, kind string) (result map[string]interface{}, err error) {
	mapping := map[string]interface{}{}
	err = json.Unmarshal([]byte(PermissionMapping), &mapping)
	if err != nil {
		log.Println("ERROR while unmarshaling PermissionMapping", err)
		return result, err
	}
	if typeMappings, ok := config.IndexTypeMapping[kind]; ok {
		if featureMappings, ok := typeMappings["features"]; ok {
			mapping["features"] = map[string]interface{}{
				"properties": featureMappings,
			}
		}
		if annotationMappings, ok := typeMappings["annotations"]; ok {
			mapping["annotations"] = map[string]interface{}{
				"properties": annotationMappings,
			}
		}
	}
	result = map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": mapping,
		},
		"settings": map[string]interface{}{
			"analysis": map[string]interface{}{
				"filter": map[string]interface{}{
					"autocomplete_filter": map[string]interface{}{
						"type":     "edge_ngram",
						"min_gram": 1,
						"max_gram": 20,
					},
					"custom_word_delimiter_filter": map[string]interface{}{
						"type":              "word_delimiter",
						"preserve_original": true,
					},
				},
				"normalizer": map[string]interface{}{
					"sortable": map[string]interface{}{
						"type": "custom",
						"filter": []string{
							"lowercase",
							"asciifolding",
						},
					},
				},
				"analyzer": map[string]interface{}{
					"custom_analyzer": map[string]interface{}{
						"type":      "custom",
						"tokenizer": "whitespace",
						"filter": []string{
							"custom_word_delimiter_filter",
							"lowercase",
							"unique",
							"autocomplete_filter",
						},
					},
					"custom_search_analyzer": map[string]interface{}{
						"type":      "custom",
						"tokenizer": "whitespace",
						"filter": []string{
							"word_delimiter",
							"lowercase",
						},
					},
				},
			},
		},
	}
	foo, err := json.Marshal(result)
	log.Println("DEBUG:", string(foo))
	return result, nil
}

type ResourceVersion struct {
	SeqNo       int64
	PrimaryTerm int64
}

type Done struct {
	ResourceKind string `json:"resource_kind"`
	ResourceId   string `json:"resource_id"`
	Handler      string `json:"handler"` // == github.com/SENERGY-Platform/permission-search
	Command      string `json:"command"` // PUT | DELETE | RIGHTS
}
