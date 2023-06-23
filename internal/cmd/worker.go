package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/zwolsman/scraping-sloth/internal/worker"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

var jobs = map[string]worker.NewWorkerFun{
	"ah": worker.NewAlbertHeijnWorker,
}

func WorkerCmd(ctx context.Context) *cobra.Command {
	var jobID string

	cmd := &cobra.Command{
		Use:   "worker",
		Args:  cobra.ExactArgs(0),
		Short: "Execute jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := pgxpool.New(cmd.Context(), os.Getenv("DATABASE_URL"))

			if err != nil {
				return fmt.Errorf("unable to connect to database: %w", err)
			}

			workerFn, ok := jobs[jobID]
			if !ok {
				return fmt.Errorf("invalid queue: %s", jobID)
			}

			logger, _ := zap.NewDevelopment(zap.AddStacktrace(zapcore.FatalLevel))
			defer logger.Sync()

			worker := workerFn(ctx, logger, db)

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

	cmd.Flags().StringVar(&jobID, "job", "", "The job to work on")

	return cmd
}
