package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/buger/jsonparser"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type ahWorker struct {
	context.Context
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

	var errGroup errgroup.Group
	startFn := func(slug string) func() error {
		return func() error {
			return a.start(slug)
		}
	}

	for _, s := range slugs {
		errGroup.Go(startFn(s))
	}

	return errGroup.Wait()
}

func (a ahWorker) start(slug string) error {
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
			return err
		}
	}
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
			ids := p.Gtins

			a.logger.Info("found product info", zap.String("title", p.Title), zap.Float64("price", price), zap.Ints("ids", ids))
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

func NewAlbertHeijnWorker(ctx context.Context, logger *zap.Logger) Worker {
	return &ahWorker{
		Context: ctx,
		logger:  logger,
	}
}
