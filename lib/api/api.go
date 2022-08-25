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

package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/permission-search/lib/api/util"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/rigthsproducer"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"reflect"
	"runtime"
	"runtime/debug"
	"time"
)

var endpoints = []func(router *httprouter.Router, config configuration.Config, query Query, p *rigthsproducer.Producer) bool{}

func Start(ctx context.Context, config configuration.Config, query Query, p *rigthsproducer.Producer) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprint(r))
		}
	}()

	timeout, err := time.ParseDuration(config.HttpServerTimeout)
	if err != nil {
		log.Println("WARNING: invalid http server timeout --> no timeouts\n", err)
		err = nil
	}

	readtimeout, err := time.ParseDuration(config.HttpServerReadTimeout)
	if err != nil {
		log.Println("WARNING: invalid http server read timeout --> no timeouts\n", err)
		err = nil
	}

	router := GetRouter(config, query, p)
	server := &http.Server{Addr: ":" + config.ServerPort, Handler: router, WriteTimeout: timeout, ReadTimeout: readtimeout}
	go func() {
		log.Println("listening on ", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			debug.PrintStack()
			config.HandleFatalError("FATAL:", err)
		}
	}()
	go func() {
		<-ctx.Done()
		log.Println("api shutdown", server.Shutdown(context.Background()))
	}()
	return
}

func GetRouter(config configuration.Config, query Query, p *rigthsproducer.Producer) http.Handler {
	router := httprouter.New()
	for _, e := range endpoints {
		if e(router, config, query, p) {
			log.Println("added endpoint: " + runtime.FuncForPC(reflect.ValueOf(e).Pointer()).Name())
		}

	}
	handler := util.NewCors(router)
	handler = util.NewLogger(handler)
	return handler
}
