package cmd

import (
	"cloud.google.com/go/pubsub"
	"context"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	common_pubsub "github.com/zwolsman/scraping-sloth/internal/common/pubsub"
	"github.com/zwolsman/scraping-sloth/internal/strategies"
	"github.com/zwolsman/scraping-sloth/internal/strategies/ah"
	"github.com/zwolsman/scraping-sloth/internal/strategies/jumbo"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
)

var distributors = map[string]strategies.NewDistributorFunc{
	"ah":    ah.NewDistributor,
	"jumbo": jumbo.NewDistributor,
}

func DistributorCmd(ctx context.Context) *cobra.Command {
	var strategy string

	cmd := &cobra.Command{
		Use:   "distributor",
		Args:  cobra.ExactArgs(0),
		Short: "Distribute jobs for the workers to consume",
		RunE: func(cmd *cobra.Command, args []string) error {
			distributorFn, ok := distributors[strategy]
			if !ok {
				return fmt.Errorf("invalid strategy: %s", strategy)
			}

			logger, _ := zap.NewDevelopment(zap.AddStacktrace(zapcore.FatalLevel))
			defer logger.Sync()

			client, err := pubsub.NewClient(ctx, "sloth")
			if err != nil {
				log.Fatal(err)
			}

			factory := common_pubsub.NewPublisherFactory(client, logger)
			distributor := distributorFn(ctx, logger, factory)

			if err := distributor.Start(); err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}

				return err
			}

			<-ctx.Done()

			distributor.Stop()
			return nil
		},
	}

	cmd.Flags().StringVar(&strategy, "strategy", "", "The strategy to apply for distributing jobs")

	return cmd
}
