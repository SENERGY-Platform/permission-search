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
	"fmt"
	"github.com/SENERGY-Platform/permission-search/lib/api/util"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	jwt_http_router "github.com/SmartEnergyPlatform/jwt-http-router"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"reflect"
	"runtime"
	"runtime/debug"
	"time"
)

var endpoints = []func(router *jwt_http_router.Router, config configuration.Config, query Query){}

func Start(ctx context.Context, config configuration.Config, query Query) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprint(r))
		}
	}()
	router := GetRouter(config, query)
	server := &http.Server{Addr: ":" + config.ServerPort, Handler: router, WriteTimeout: 10 * time.Second, ReadTimeout: 2 * time.Second, ReadHeaderTimeout: 2 * time.Second}
	go func() {
		log.Println("listening on ", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			debug.PrintStack()
			log.Fatal("FATAL:", err)
		}
	}()
	go func() {
		<-ctx.Done()
		log.Println("api shutdown", server.Shutdown(context.Background()))
	}()
	return
}

func GetRouter(config configuration.Config, query Query) http.Handler {
	jwt := jwt_http_router.New(jwt_http_router.JwtConfig{ForceAuth: true, ForceUser: true})
	for _, e := range endpoints {
		log.Println("add endpoint: " + runtime.FuncForPC(reflect.ValueOf(e).Pointer()).Name())
		e(jwt, config, query)
	}
	handler := util.NewCors(jwt)
	handler = util.NewLogger(handler)
	return handler
}
