package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"io"
	"net/http"
)

type jumboWorker struct {
	context.Context
	logger *zap.Logger
	db     *pgxpool.Pool
}

func (j jumboWorker) Start() error {
	const pageSize = 24
	var limit *int

	for offset := 0; limit == nil || offset < *limit; offset += pageSize {
		payload := json.RawMessage(fmt.Sprintf(`
{"query":"query SearchProducts($input: ProductSearchInput!) {\n\tsearchProducts(input: $input) {\n\t\tstart\n\t\tcount\n\t\tproducts {\n\t\t\tbrand\n\t\t\tean\n\t\t\ttitle\n\t\t\tprices: price {\n\t\t\t\tprice\n\t\t\t\tpromoPrice\n\t\t\t}\n\t\t}\n\t\t__typename\n\t}\n}\n","operationName":"SearchProducts","variables":{"input":{"searchType":"category","searchTerms":"producten","offSet":%d,"currentUrl":"https://www.jumbo.com/producten/","previousUrl":"https://www.jumbo.com/producten/"}}}
`, offset))

		j.logger.Debug("fetching new page", zap.Int("offset", offset))
		req, err := http.NewRequest("POST", jumboUrl, bytes.NewBuffer(payload))
		if err != nil {
			return err
		}

		req.Header.Add("Origin", "https://www.jumbo.com")
		req.Header.Add("Referer", fmt.Sprintf("https://www.jumbo.com/producten/?offSet=%d", offset))
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Accept", "application/json")
		req.Header.Add("User-Agent", "insomnia/2023.2.2")
		req.WithContext(j.Context)

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}

		data, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		if limit == nil {
			limit = getLimit(data)
			if limit != nil {
				j.logger.Debug("found limit", zap.Int("limit", *limit))
			} else {
				j.logger.Debug("limit not parsed")
			}
		}

		if err := j.parseResponse(data); err != nil {
			if !errors.Is(err, context.Canceled) {
				j.logger.Error("failed parsing response", zap.Error(err), zap.ByteString("payload", data))
			}
			return err
		}
	}

	j.logger.Info("finished processing")
	return nil
}

func (j jumboWorker) parseResponse(data []byte) error {
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

		_, err := j.db.Exec(j.Context, "INSERT INTO prices(gtin, title, shop, price) VALUES($1, $2, 'JUMBO', $3) ON CONFLICT DO NOTHING", product.Ean, product.Title, price)
		if err != nil {
			return err
		}
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

func (j jumboWorker) Stop() {
	j.logger.Info("stopping jumbo worker")
}

func NewJumboWorker(ctx context.Context, logger *zap.Logger, db *pgxpool.Pool) Worker {
	return &jumboWorker{
		Context: ctx,
		logger:  logger,
		db:      db,
	}
}
