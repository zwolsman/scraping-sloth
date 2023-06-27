package jumbo

import (
	"context"
	"errors"
	"github.com/buger/jsonparser"
	"github.com/zwolsman/scraping-sloth/internal/common/pubsub"
	"github.com/zwolsman/scraping-sloth/internal/strategies"
	"go.uber.org/zap"
	"io"
	"net/http"
)

type distributor struct {
	context.Context
	topics pubsub.PublisherCreator
	logger *zap.Logger
}

func NewDistributor(ctx context.Context, logger *zap.Logger, topics pubsub.PublisherCreator) strategies.Distributor {
	return &distributor{
		Context: ctx,
		topics:  topics,
		logger:  logger,
	}
}

func (d *distributor) Start() error {
	d.logger.Info("starting jumbo distributor")
	publisher, err := d.topics.Create(d, topic)
	if err != nil {
		return err
	}

	req, err := createRequest(d, 0, []byte(countQuery))
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	limit := getLimit(data)
	if limit == nil {
		return errors.New("could not fetch limit")
	}

	for offset := 0; offset < *limit; offset += pageSize {
		err = publisher.Publish(d, &message{
			Offset: offset,
		})
	}

	return nil
}

func getLimit(data []byte) *int {
	n, err := jsonparser.GetInt(data, "data", "searchProducts", "count")
	if err != nil {
		return nil
	}

	limit := int(n)
	return &limit
}

func (d *distributor) Stop() {
	d.logger.Info("shutting down jumbo distributor")
}
