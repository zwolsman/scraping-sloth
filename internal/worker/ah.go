package worker

import (
	"context"
	"go.uber.org/zap"
)

func NewAlbertHeijnWorker(_ context.Context, logger *zap.Logger) {
	logger.Info("hello")
}
