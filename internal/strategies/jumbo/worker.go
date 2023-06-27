package jumbo

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

func (w *worker) Handle(ctx context.Context, raw []byte) error {
	var msg message
	err := json.Unmarshal(raw, &msg)
	if err != nil {
		return err
	}

	payload := json.RawMessage(fmt.Sprintf(productQueryFormat, msg.Offset))
	req, err := createRequest(ctx, msg.Offset, payload)
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
		return err
	}

	return nil
}

func NewWorker(ctx context.Context, logger *zap.Logger, db *pgxpool.Pool, subs pubsub.SubscriptionCreator) strategies.Worker {
	return &worker{
		ctx,
		logger,
		db,
		subs,
	}
}

func (w *worker) process(data []byte) error {
	type page struct {
		Data struct {
			SearchProducts struct {
				Products []struct {
					Brand  *string `json:"brand"`
					Ean    string  `json:"ean"`
					Title  string  `json:"title"`
					Prices struct {
						Price      int  `json:"price"`
						PromoPrice *int `json:"promoPrice"`
					} `json:"prices"`
				} `json:"products"`
			} `json:"searchProducts"`
		} `json:"data"`
	}
	var p page
	err := json.Unmarshal(data, &p)
	if err != nil {
		return err
	}

	for _, product := range p.Data.SearchProducts.Products {
		price := float64(product.Prices.Price)
		if promo := product.Prices.PromoPrice; promo != nil {
			price = float64(*promo)
		}

		//price is in cents but the data is stored as eur
		price /= 100

		_, err := w.db.Exec(w, "INSERT INTO prices(gtin, title, shop, price) VALUES($1, $2, 'JUMBO', $3) ON CONFLICT DO NOTHING", product.Ean, product.Title, price)
		if err != nil {
			return err
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
	w.logger.Info("stopping jumbo worker")
}
