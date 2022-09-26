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
	"log"
	"net/url"
	"strings"
)

const Seperator = "$"

func DecodeModifierParameter(parameter string) (result map[string][]string, err error) {
	return url.ParseQuery(parameter)
}

func EncodeModifierParameter(parameter map[string][]string) (result string) {
	return url.Values(parameter).Encode()
}

func SplitModifier(id string) (pureId string, modifier map[string][]string) {
	parts := strings.SplitN(id, Seperator, 2)
	pureId = parts[0]
	if len(parts) < 2 {
		return
	}
	var err error
	modifier, err = DecodeModifierParameter(parts[1])
	if err != nil {
		log.Println("WARNING: unable to parse modifier parts as Modifier --> ignore modifiers")
		modifier = nil
		return
	}
	return
}
