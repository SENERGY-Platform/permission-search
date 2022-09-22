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

package modifier

import (
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"log"
	"runtime/debug"
)

func (this *Modifier) checkModifyConditions(entry map[string]interface{}, conditions []configuration.ModifierCondition) bool {
	for _, condition := range conditions {
		switch condition.Operator {
		case "not_empty":
			value, err := jsonPathGetFirst(entry, condition.Path)
			if err != nil {
				log.Println("ERROR:", err)
				debug.PrintStack()
				return false
			}
			if value == nil {
				return false
			}
		}
	}
	return true
}
