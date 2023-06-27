package ah

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zwolsman/scraping-sloth/internal/common/pubsub"
	"github.com/zwolsman/scraping-sloth/internal/strategies"
	"go.uber.org/zap"
	"io"
	"net/http"
)

type worker struct {
	context.Context

	logger *zap.Logger
	db     *pgxpool.Pool
	subs   pubsub.SubscriptionCreator
}

func NewWorker(ctx context.Context, logger *zap.Logger, db *pgxpool.Pool, subs pubsub.SubscriptionCreator) strategies.Worker {
	return &worker{
		ctx,
		logger,
		db,
		subs,
	}
}

func (w *worker) Handle(ctx context.Context, raw []byte) error {
	var msg message
	err := json.Unmarshal(raw, &msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", msg.Target, nil)
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

	err = w.process(data)
	if err != nil {
		w.logger.Info("failed to process", zap.ByteString("response", data))
		return fmt.Errorf("failed to process response: %w", err)
	}

	w.logger.Debug("finished processing", zap.String("target", msg.Target))
	return nil
}

func (w *worker) process(data []byte) error {
	type page struct {
		Cards []struct {
			Products []struct {
				Id     int    `json:"id"`
				Title  string `json:"title"`
				Link   string `json:"link"`
				Images []struct {
					Height int    `json:"height"`
					Width  int    `json:"width"`
					Title  string `json:"title"`
					Url    string `json:"url"`
					Ratio  string `json:"ratio"`
				} `json:"images"`
				Price struct {
					Now float64 `json:"now"`
				} `json:"price"`
				Brand string `json:"brand"`
				Gtins []int  `json:"gtins"`
			} `json:"products"`
		} `json:"cards"`
	}
	var p page

	err := json.Unmarshal(data, &p)
	if err != nil {
		return err
	}

	for _, c := range p.Cards {
		for _, p := range c.Products {
			price := p.Price.Now

			for _, gtin := range p.Gtins {
				_, err := w.db.Exec(w.Context, "INSERT INTO prices(gtin, title, shop, price) VALUES($1, $2, 'AH', $3) ON CONFLICT DO NOTHING", gtin, p.Title, price)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (w *worker) Start() error {
	sub, err := w.subs.Create(w.Context, subscription, w)
	if err != nil {
		return err
	}

	return sub.Start(w.Context)
}

func (w *worker) Stop() {
	w.logger.Info("stopping ah worker")
}
