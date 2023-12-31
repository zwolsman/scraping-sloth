package cmd

import (
	"cloud.google.com/go/pubsub"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	common_pubsub "github.com/zwolsman/scraping-sloth/internal/common/pubsub"
	"github.com/zwolsman/scraping-sloth/internal/strategies"
	"github.com/zwolsman/scraping-sloth/internal/strategies/ah"
	"github.com/zwolsman/scraping-sloth/internal/strategies/jumbo"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"os"
)

var workers = map[string]strategies.NewWorkerFun{
	"ah":    ah.NewWorker,
	"jumbo": jumbo.NewWorker,
}

func WorkerCmd(ctx context.Context) *cobra.Command {
	var strategy string

	cmd := &cobra.Command{
		Use:   "worker",
		Args:  cobra.ExactArgs(0),
		Short: "Execute jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := pgxpool.New(cmd.Context(), os.Getenv("DATABASE_URL"))

			if err != nil {
				return fmt.Errorf("unable to connect to database: %w", err)
			}

			workerFn, ok := workers[strategy]
			if !ok {
				return fmt.Errorf("invalid strategy: %s", strategy)
			}

			logger, _ := zap.NewDevelopment(zap.AddStacktrace(zapcore.FatalLevel))
			defer logger.Sync()

			client, err := pubsub.NewClient(ctx, "sloth")
			if err != nil {
				log.Fatal(err)
			}

			subs := common_pubsub.NewSubscriptionFactory(client, logger)

			worker := workerFn(ctx, logger, db, subs)

			if err := worker.Start(); err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}

				return err
			}

			<-ctx.Done()

			worker.Stop()
			return nil
		},
	}

	cmd.Flags().StringVar(&strategy, "strategy", "", "The strategy to process jobs for")

	return cmd
}
