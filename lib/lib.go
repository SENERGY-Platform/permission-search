package lib

import (
	"context"
	"errors"
	"github.com/SENERGY-Platform/permission-search/lib/api"
	"github.com/SENERGY-Platform/permission-search/lib/configuration"
	"github.com/SENERGY-Platform/permission-search/lib/query"
	"github.com/SENERGY-Platform/permission-search/lib/worker"
)

type Mode string

const (
	Query      Mode = "query"
	Worker     Mode = "worker"
	Standalone Mode = "standalone"
)

func GetMode(s string) (mode Mode, err error) {
	switch Mode(s) {
	case Query:
		return Query, nil
	case Worker:
		return Worker, nil
	case Standalone:
		return Standalone, nil
	default:
		return "", errors.New("unknown mode")
	}
}

func Start(parentctx context.Context, config configuration.Config, mode Mode) (err error) {
	_, _, err = StartGetComponents(parentctx, config, mode)
	return
}

func StartGetComponents(parentctx context.Context, config configuration.Config, mode Mode) (q *query.Query, w *worker.Worker, err error) {
	ctx, cancel := context.WithCancel(parentctx)
	defer func() {
		if err != nil {
			cancel()
		}
	}()
	q, err = query.New(config)
	if err != nil {
		return q, w, err
	}
	if mode == Query || mode == Standalone {
		err = api.Start(ctx, config, q)
		if err != nil {
			return q, w, err
		}
	}

	if mode == Worker || mode == Standalone {
		w = worker.New(config, q)
		err = worker.InitEventHandling(ctx, config, w)
		if err != nil {
			return q, w, err
		}
	}
	return q, w, nil
}
