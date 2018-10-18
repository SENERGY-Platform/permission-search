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
	"log"
)

const ElasticPermissionType = "resource"

func (entry *Entry) setDefaultPermissions(kind string, owner string) {
	if owner != "" {
		entry.AdminUsers = []string{owner}
		entry.ReadUsers = []string{owner}
		entry.WriteUsers = []string{owner}
		entry.ExecuteUsers = []string{owner}
	}
	for group, rights := range Config.Resources[kind].InitialGroupRights {
		entry.addGroupRights(group, rights)
	}
	return
}

func (entry *Entry) addUserRights(user string, rights string) {
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

func (entry *Entry) removeUserRights(user string) {
	entry.AdminUsers = listRemove(entry.AdminUsers, user)
	entry.ReadUsers = listRemove(entry.ReadUsers, user)
	entry.WriteUsers = listRemove(entry.WriteUsers, user)
	entry.ExecuteUsers = listRemove(entry.ExecuteUsers, user)
}

func (entry *Entry) addGroupRights(group string, rights string) {
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

func (entry *Entry) removeGroupRights(group string) {
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
	Command string `json:"command"`
	Id      string `json:"id"`
	Owner   string `json:"owner"`
}

type ResourceRights struct {
	ResourceId  string                 `json:"resource_id"`
	Features    map[string]interface{} `json:"features"`
	UserRights  map[string]Right       `json:"user_rights"`
	GroupRights map[string]Right       `json:"group_rights"`
	Creator     string                 `json:"creator"`
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

func (this *Entry) SetResourceRights(rights ResourceRights) {
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

const ElasticPermissionMapping = `{
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
	"feature_search": {"type": "text", "analyzer": "autocomplete", "search_analyzer": "standard"}
}`

func createMapping(kind string) (result map[string]map[string]map[string]map[string]interface{}, err error) {
	mapping := map[string]interface{}{}
	err = json.Unmarshal([]byte(ElasticPermissionMapping), &mapping)
	if err != nil {
		log.Println("ERROR while unmarshaling ElasticPermissionMapping", err)
		return result, err
	}
	if featureMappings, ok := Config.ElasticMapping[kind]; ok {
		mapping["features"] = map[string]interface{}{
			"properties": featureMappings,
		}
	}
	result = map[string]map[string]map[string]map[string]interface{}{
		"mappings": {
			ElasticPermissionType: {
				"properties": mapping,
			},
		},
		"settings": {
			"analysis": {
				"filter": {
					"autocomplete_filter": map[string]interface{}{
						"type":     "edge_ngram",
						"min_gram": 1,
						"max_gram": 20,
					},
				},
				"analyzer": {
					"autocomplete": map[string]interface{}{
						"type":      "custom",
						"tokenizer": "standard",
						"filter": []string{
							"lowercase",
							"autocomplete_filter",
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
