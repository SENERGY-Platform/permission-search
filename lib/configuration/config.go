package configuration

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
	Features           []Feature            `json:"features"`
	Annotations        map[string][]Feature `json:"annotations"`
	InitialGroupRights map[string]string    `json:"initial_group_rights"`
}

type ConfigStruct struct {
	Debug bool `json:"debug"`

	ServerPort string `json:"server_port"`
	LogLevel   string `json:"log_level"`

	KafkaUrl string `json:"kafka_url"`

	PermTopic string `json:"perm_topic"`
	UserTopic string `json:"user_topic"`

	ElasticUrl     string                                       `json:"elastic_url"`
	ElasticRetry   int64                                        `json:"elastic_retry"`
	ElasticMapping map[string]map[string]map[string]interface{} `json:"elastic_mapping"`

	JwtPubRsa string `json:"jwt_pub_rsa"`
	ForceUser string `json:"force_user"`
	ForceAuth string `json:"force_auth"`

	Resources               map[string]ResourceConfig `json:"resources"`
	ResourceList            []string                  `json:"-"`
	AnnotationResourceIndex map[string][]string       `json:"-"`

	GroupId string `json:"group_id"`
}

type Config = *ConfigStruct

func LoadConfig(location string) (config Config, err error) {
	file, error := os.Open(location)
	if error != nil {
		log.Println("error on config load: ", error)
		return config, error
	}
	decoder := json.NewDecoder(file)
	error = decoder.Decode(&config)
	if error != nil {
		log.Println("invalid config json: ", error)
		return config, error
	}
	HandleEnvironmentVars(config)
	config.ResourceList = getResourceList(config)
	config.AnnotationResourceIndex = getAnnotationResourceIndex(config)
	return config, nil
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
func HandleEnvironmentVars(config Config) {
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
			if configValue.FieldByName(fieldName).Kind() == reflect.Float64 {
				f, _ := strconv.ParseFloat(envValue, 64)
				configValue.FieldByName(fieldName).SetFloat(f)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.String {
				configValue.FieldByName(fieldName).SetString(envValue)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Bool {
				b, _ := strconv.ParseBool(envValue)
				configValue.FieldByName(fieldName).SetBool(b)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Slice {
				val := []string{}
				for _, element := range strings.Split(envValue, ",") {
					val = append(val, strings.TrimSpace(element))
				}
				configValue.FieldByName(fieldName).Set(reflect.ValueOf(val))
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Map {
				value := map[string]string{}
				for _, element := range strings.Split(envValue, ",") {
					keyVal := strings.Split(element, ":")
					key := strings.TrimSpace(keyVal[0])
					val := strings.TrimSpace(keyVal[1])
					value[key] = val
				}
				configValue.FieldByName(fieldName).Set(reflect.ValueOf(value))
			}
		}
	}
}
