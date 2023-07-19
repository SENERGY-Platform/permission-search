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
	"context"
	"errors"
	"flag"
	"github.com/SENERGY-Platform/permission-search/lib"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/opensearchclient"
	"github.com/SENERGY-Platform/permission-search/lib/replay"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	configLocation := flag.String("config", "config.json", "configuration file")
	modeFlag := flag.String("mode", "standalone", "may be query, worker or standalone")
	flag.Parse()

	mode, err := lib.GetMode(*modeFlag)
	if err != nil {
		log.Fatal(err)
	}

	config, err := configuration.LoadConfig(*configLocation)
	if err != nil {
		log.Fatal(err)
	}

	args := flag.Args()
	if len(args) > 0 {
		log.Println("handle cli args", args)
		err := HandleCli(config, args)
		if err != nil {
			log.Fatal("FATAL:", err)
		}
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	err = lib.Start(ctx, config, mode)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
		sig := <-shutdown
		log.Println("received shutdown signal", sig)
		cancel()
	}()

	<-ctx.Done()                //waiting for context end; may happen by shutdown signal
	time.Sleep(5 * time.Second) //give go routines time for cleanup and last messages
}

func HandleCli(config configuration.Config, args []string) error {
	if len(args) == 0 {
		return errors.New("no command in args")
	}
	switch args[0] {
	case "replay-permissions":
		replay.ReplayPermissions(config, args[1:])
		return nil
	case "update-indexes":
		resources := args[1:]
		if len(resources) == 0 {
			resources = config.ResourceList
		}
		return opensearchclient.UpdateIndexes(config, resources...)
	default:
		return errors.New("unknown command: " + args[0])
	}
}
