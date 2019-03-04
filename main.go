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

package main

import (
	"flag"
	"log"
	"time"

	"github.com/SENERGY-Platform/permission-search/lib"
)

func main() {
	configLocation := flag.String("config", "config.json", "configuration file")
	flag.Parse()

	err := lib.LoadConfig(*configLocation)
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Duration(lib.Config.AmqpReconnectTimeout) * time.Second)

	if lib.Config.DbInitOnly == "true" {
		lib.GetClient()
	} else {
		if lib.Config.InitialGroupRightsUpdate == "true" {
			lib.UpdateInitialGroupRights()
		}
		if lib.Config.ConsumptionPause == "true" {
			log.Println("pause event consumption")
		} else {
			lib.InitEventHandling()
		}
		lib.StartApi()
	}
}
