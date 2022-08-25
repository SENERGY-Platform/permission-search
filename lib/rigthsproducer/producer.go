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

package rigthsproducer

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/worker/kafka"
	"net/http"
)

type Producer struct {
	writers map[string]*kafka.Producer
}

func New(ctx context.Context, config configuration.Config) (result *Producer, err error) {
	result = &Producer{
		writers: map[string]*kafka.Producer{},
	}
	for _, topic := range config.ResourceList {
		result.writers[topic], err = kafka.NewProducerWithKeySeparationBalancer(ctx, config.KafkaUrl, topic, config.Debug)
		if err != nil {
			return result, err
		}
	}
	return result, nil
}

func (this *Producer) SetResourceRights(resource string, id string, rights model.ResourceRightsBase, key string) (err error, code int) {
	cmd := model.CommandWithRights{
		Command: "RIGHTS",
		Id:      id,
		Rights:  &rights,
	}
	if writer, ok := this.writers[resource]; ok {
		var temp []byte
		temp, err = json.Marshal(cmd)
		if err != nil {
			return err, http.StatusInternalServerError
		}
		if key == "" {
			key = id + "/rights"
		}
		err = writer.Produce(key, temp)
		if err != nil {
			return err, http.StatusInternalServerError
		}
	} else {
		return errors.New("unknown resource"), http.StatusNotFound
	}
	return nil, http.StatusOK
}
