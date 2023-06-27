package strategies

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zwolsman/scraping-sloth/internal/common/pubsub"
	"go.uber.org/zap"
	"io"
	"net/http"
)

const (
	ahUrlFormat   = "https://www.ah.nl/zoeken/api/products/search?page=%d&size=%d&taxonomySlug=%s"
	jumboUrl      = "https://www.jumbo.com/api/graphql"
	plusUrlFormat = "https://www.plus.nl/INTERSHOP/web/WFS/PLUS-website-Site/nl_NL/-/EUR/ViewTWSearch-ProductPaging?PageNumber=%d&PageSize=%d&SearchTerm="
)

type NewWorkerFun func(context.Context, *zap.Logger, *pgxpool.Pool, pubsub.SubscriptionCreator) Worker

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
