package worker

import (
	"context"
	"go.uber.org/zap"
)

type NewWorkerFun func(context.Context, *zap.Logger)
