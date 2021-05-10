/*
 * Copyright 2020 InfAI (CC SES)
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

package kafka

import (
	"github.com/SENERGY-Platform/permission-search/lib/worker/kafka/topicconfig"
)

func InitTopic(bootstrapUrl string, topics ...string) (err error) {
	for _, topic := range topics {
		err = topicconfig.Ensure(bootstrapUrl, topic, map[string]string{
			"retention.ms":              "-1",
			"retention.bytes":           "-1",
			"cleanup.policy":            "compact",
			"delete.retention.ms":       "86400000",
			"segment.ms":                "604800000",
			"min.cleanable.dirty.ratio": "0.1",
		})
		if err != nil {
			return err
		}
	}
	return nil
}
