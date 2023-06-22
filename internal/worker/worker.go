package worker

import (
	"context"
	"go.uber.org/zap"
	"io"
	"net/http"
)

type NewWorkerFun func(context.Context, *zap.Logger) Worker

type Worker interface {
	Start() error
	Stop()
}

func get(url string) ([]byte, error) {
	res, err := http.DefaultClient.Get(url)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(res.Body)
}
