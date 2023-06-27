package ah

import (
	"context"
	"fmt"
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
	d.logger.Info("setting up publish topic")
	publisher, err := d.topics.Create(d, topic)
	if err != nil {
		return fmt.Errorf("failed to create publisher: %w", err)
	}

	// Run it
	if err := d.run(publisher); err != nil {
		return err
	}

	d.logger.Info("done")
	return nil
}

func (d *distributor) run(publisher pubsub.Publisher) error {
	url := fmt.Sprintf(distributorUrlFormat, pageSize)

	req, err := http.NewRequestWithContext(d, "GET", url, nil)
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

	pages, err := jsonparser.GetInt(data, "page", "totalPages")

	for n := 0; n < int(pages-1); n++ {
		err = publisher.Publish(d, &message{
			Target: url + fmt.Sprintf("&page=%d", n+1),
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func (d *distributor) Stop() {
	d.logger.Info("shutting down", zap.String("distributor", "ah"))
}
