package worker

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"io"
	"net/http"
)

const (
	ahUrlFormat = "https://www.ah.nl/zoeken/api/products/search?page=%d&size=%d&taxonomySlug=%s"
	jumboUrl    = "https://www.jumbo.com/api/graphql"
)

type NewWorkerFun func(context.Context, *zap.Logger, *pgxpool.Pool) Worker

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
