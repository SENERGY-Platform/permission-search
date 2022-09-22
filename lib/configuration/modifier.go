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

package configuration

type ResultModifier struct {
	Modifier string            `json:"modifier"`
	Template *TemplateModifier `json:"template"`
}

type TemplateModifier struct {
	Path       string                       `json:"path"`
	Value      string                       `json:"value"`
	Conditions []ModifierCondition          `json:"conditions"`
	References map[string]ModifierReference `json:"references"`
}

type ModifierCondition struct {
	Path     string `json:"path"`
	Operator string `json:"operator"`
}

type ModifierReference struct {
	Resource   string      `json:"resource"`
	ResourceId string      `json:"resource_id"`
	Path       string      `json:"path"`
	Default    interface{} `json:"default"`
}
