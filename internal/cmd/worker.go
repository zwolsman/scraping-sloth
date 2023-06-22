package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/zwolsman/scraping-sloth/internal/worker"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

			workerFn, ok := jobs[jobID]
			if !ok {
				return fmt.Errorf("invalid queue: %s", jobID)
			}

			logger, _ := zap.NewDevelopment(zap.AddStacktrace(zapcore.FatalLevel))
			defer logger.Sync()
			workerFn(ctx, logger)

			<-ctx.Done()
			return nil
		},
	}

	cmd.Flags().StringVar(&jobID, "job", "", "The job to work on")

	return cmd
}
