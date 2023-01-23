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

package lib

import (
	"context"
	"errors"
	"github.com/SENERGY-Platform/permission-search/lib/api"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/query"
	"github.com/SENERGY-Platform/permission-search/lib/rigthsproducer"
	"github.com/SENERGY-Platform/permission-search/lib/worker"
)

type Mode string

const (
	Query      Mode = "query"
	Command    Mode = "command"
	Worker     Mode = "worker"
	Standalone Mode = "standalone"
)

func GetMode(s string) (mode Mode, err error) {
	switch Mode(s) {
	case Query:
		return Query, nil
	case Command:
		return Command, nil
	case Worker:
		return Worker, nil
	case Standalone:
		return Standalone, nil
	default:
		return "", errors.New("unknown mode")
	}
}

func Start(parentctx context.Context, config configuration.Config, mode Mode) (err error) {
	_, _, _, err = StartGetComponents(parentctx, config, mode)
	return
}

func StartGetComponents(parentctx context.Context, config configuration.Config, mode Mode) (q *query.Query, p *rigthsproducer.Producer, w *worker.Worker, err error) {
	ctx, cancel := context.WithCancel(parentctx)
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	q, err = query.New(config)
	if err != nil {
		return q, p, w, err
	}

	if mode == Command || mode == Standalone {
		p, err = rigthsproducer.New(ctx, config)
		if err != nil {
			return q, p, w, err
		}
	}

	if mode == Query || mode == Command || mode == Standalone {
		err = api.Start(ctx, config, q, p)
		if err != nil {
			return q, p, w, err
		}
	}

	if mode == Worker || mode == Standalone {
		w, err = worker.New(ctx, config, q)
		if err != nil {
			return q, p, w, err
		}
		err = worker.InitEventHandling(ctx, config, w)
		if err != nil {
			return q, p, w, err
		}
	}
	return q, p, w, nil
}
