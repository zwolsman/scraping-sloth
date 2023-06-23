package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type ahWorker struct {
	context.Context
	db     *pgxpool.Pool
	logger *zap.Logger
}

const pageSize = 100
const urlFormat = "https://www.ah.nl/zoeken/api/products/search?page=%d&size=%d&taxonomySlug=%s"

var slugs = []string{
	"aardappel-groente-fruit",
	"baby-en-kind",
	"bakkerij-en-banket",
	"bier-en-aperitieven",
	"diepvries",
	"drogisterij",
	"frisdrank-sappen-koffie-thee",
	"huisdier",
	"huishouden",
	"kaas-vleeswaren-tapas",
	"koken-tafelen-vrije-tijd",
	"ontbijtgranen-en-beleg",
	"pasta-rijst-en-wereldkeuken",
	"salades-pizza-maaltijden",
	"snoep-koek-chips-en-chocolade",
	"soepen-sauzen-kruiden-olie",
	"sport-en-dieetvoeding",
	"tussendoortjes",
	"vlees-kip-vis-vega",
	"wijn-en-bubbels",
	"zuivel-plantaardig-en-eieren",
}

func (a ahWorker) Start() error {
	a.logger.Info("starting ah worker")
	for _, s := range slugs {
		if err := a.scrape(s); err != nil {
			return err
		}
	}

	a.logger.Info("finished processing all slugs", zap.Strings("slugs", slugs))
	return nil
}

func (a ahWorker) scrape(slug string) error {
	url := fmt.Sprintf(urlFormat, 1, pageSize, slug)
	a.logger.Debug("fetching total of pages", zap.String("url", url))
	data, err := get(url)

	pages, err := getTotalPages(data)
	if err != nil {
		return err
	}

	a.logger.Info("got total number of pages", zap.Int("pages", pages))

	// Saving original response
	err = a.parseResponse(data)
	if err != nil {
		return err
	}

	for i := 2; i < pages; i++ {
		url = fmt.Sprintf(urlFormat, i, pageSize, slug)
		a.logger.Debug("fetching new page", zap.String("url", url))

		data, err = get(url)
		if err != nil {
			return err
		}

		err = a.parseResponse(data)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				a.logger.Error("failed parsing response", zap.Error(err), zap.ByteString("payload", data))
			}
			return err
		}
	}

	a.logger.Info("finished processing slug", zap.String("slug", slug), zap.Int("totalPages", pages))
	return nil
}

func (a ahWorker) parseResponse(data []byte) error {
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
			//a.logger.Debug("found product info", zap.String("title", p.Title), zap.Float64("price", price), zap.Ints("ids", p.Gtins))

			for _, gtin := range p.Gtins {
				_, err := a.db.Exec(a.Context, "INSERT INTO prices(gtin, title, shop, price) VALUES($1, $2, 'AH', $3) ON CONFLICT DO NOTHING", gtin, p.Title, price)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func getTotalPages(data []byte) (int, error) {
	v, err := jsonparser.GetFloat(data, "page", "totalPages")
	if err != nil {
		return -1, err
	}

	return int(v), nil
}

func (a ahWorker) Stop() {
	a.logger.Info("stopping ah worker")
}

func NewAlbertHeijnWorker(ctx context.Context, logger *zap.Logger, db *pgxpool.Pool) Worker {
	return &ahWorker{
		Context: ctx,
		logger:  logger,
		db:      db,
	}
}
