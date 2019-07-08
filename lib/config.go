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
	"fmt"
	"log"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type Feature struct {
	Name string
	Path string
}

type ResourceConfig struct {
	Features              []Feature
	InitialGroupRights    map[string]string
	SearchFallbackFeature string
}

type ConfigStruct struct {
	ServerPort string
	LogLevel   string

	ZookeeperUrl	string

	PermTopic string
	UserTopic string

	ElasticUrl     string
	ElasticRetry   int64
	ElasticMapping map[string]map[string]interface{}

	JwtPubRsa string
	ForceUser string
	ForceAuth string

	Resources    map[string]ResourceConfig
	ResourceList []string `json:"-"`

	InitialGroupRightsUpdate string

	ConsumptionPause string

	DbInitOnly string
	GroupId string
}

type ConfigType *ConfigStruct

var Config ConfigType

func LoadConfig(location string) error {
	file, error := os.Open(location)
	if error != nil {
		log.Println("error on config load: ", error)
		return error
	}
	decoder := json.NewDecoder(file)
	configuration := ConfigStruct{}
	error = decoder.Decode(&configuration)
	if error != nil {
		log.Println("invalid config json: ", error)
		return error
	}
	HandleEnvironmentVars(&configuration)
	Config = &configuration
	Config.ResourceList = getResourceList(Config)
	return nil
}

var camel = regexp.MustCompile("(^[^A-Z]*|[A-Z]*)([A-Z][^A-Z]+|$)")

func fieldNameToEnvName(s string) string {
	var a []string
	for _, sub := range camel.FindAllStringSubmatch(s, -1) {
		if sub[1] != "" {
			a = append(a, sub[1])
		}
		if sub[2] != "" {
			a = append(a, sub[2])
		}
	}
	return strings.ToUpper(strings.Join(a, "_"))
}

// preparations for docker
func HandleEnvironmentVars(config ConfigType) {
	configValue := reflect.Indirect(reflect.ValueOf(config))
	configType := configValue.Type()
	for index := 0; index < configType.NumField(); index++ {
		fieldName := configType.Field(index).Name
		envName := fieldNameToEnvName(fieldName)
		envValue := os.Getenv(envName)
		if envValue != "" {
			fmt.Println("use environment variable: ", envName, " = ", envValue)
			if configValue.FieldByName(fieldName).Kind() == reflect.Int64 {
				i, _ := strconv.ParseInt(envValue, 10, 64)
				configValue.FieldByName(fieldName).SetInt(i)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.String {
				configValue.FieldByName(fieldName).SetString(envValue)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Slice {
				val := []string{}
				for _, element := range strings.Split(envValue, ",") {
					val = append(val, strings.TrimSpace(element))
				}
				configValue.FieldByName(fieldName).Set(reflect.ValueOf(val))
			}
		}
	}
}

func getResourceList(c ConfigType) (result []string) {
	for resource := range c.Resources {
		result = append(result, resource)
	}
	return
}
