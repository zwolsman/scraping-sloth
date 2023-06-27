package strategies

import (
	"context"
	"github.com/zwolsman/scraping-sloth/internal/common/pubsub"
	"go.uber.org/zap"
)

type NewDistributorFunc func(context.Context, *zap.Logger, pubsub.PublisherCreator) Distributor

type Distributor interface {
	Start() error
	Stop()
}
