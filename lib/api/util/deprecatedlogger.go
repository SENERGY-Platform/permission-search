/*
 * Copyright 2019 InfAI (CC SES)
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

package util

import (
	"log"
	"net/http"
	"os"
)

func NewDeprecatedRespHeaderLogger(handler http.Handler, filepath string) (http.Handler, error) {
	result := &DeprecatedRespHeader{handler: handler}
	if filepath != "" && filepath != "-" {
		logFile, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		result.fileLoger = log.New(logFile, "", log.LstdFlags)
	}

	return result, nil
}

type DeprecatedRespHeader struct {
	handler   http.Handler
	fileLoger *log.Logger
}

func (this *DeprecatedRespHeader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if this.handler != nil {
		this.handler.ServeHTTP(w, r)
		this.log(w, r)
	} else {
		http.Error(w, "Forbidden", 403)
	}
}

func (this *DeprecatedRespHeader) log(response http.ResponseWriter, request *http.Request) {
	if this.condition(response) {
		method := request.Method
		path := request.URL
		remote := request.RemoteAddr
		if this.fileLoger != nil {
			this.fileLoger.Printf("%v [%v] %v \n", remote, method, path)
		}
		log.Printf("[DEPRECATED-CALL] %v [%v] %v \n", remote, method, path)
	}
}

func (this *DeprecatedRespHeader) condition(response http.ResponseWriter) bool {
	return response.Header().Get("Deprecation") != ""
}
