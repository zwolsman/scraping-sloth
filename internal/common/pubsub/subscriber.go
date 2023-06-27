package pubsub

import (
	"cloud.google.com/go/pubsub"
	"context"
	"fmt"
	"go.uber.org/zap"
)

type SubscriptionCreator interface {
	Create(context.Context, string, Handler) (Subscription, error)
}

type Handler interface {
	Handle(context.Context, []byte) error
}

type Subscription interface {
	Start(ctx context.Context) error
}

type subscriptionFactory struct {
	cli    *pubsub.Client
	logger *zap.Logger
}

func NewSubscriptionFactory(cli *pubsub.Client, logger *zap.Logger) SubscriptionCreator {
	return &subscriptionFactory{cli, logger}
}

func (s subscriptionFactory) Create(ctx context.Context, subscriptionID string, h Handler) (Subscription, error) {
	sub := s.cli.Subscription(subscriptionID)

	ok, err := sub.Exists(ctx)
	if err != nil {
		return nil, err
	}
	s.logger.Debug("check subscription existing", zap.Bool("exists", ok), zap.String("subscription", subscriptionID))
	if !ok {
		return nil, fmt.Errorf("subscription does not exist: %s", subscriptionID)
	}

	sub.ReceiveSettings = pubsub.ReceiveSettings{
		MaxExtension:           pubsub.DefaultReceiveSettings.MaxExtension,
		MaxExtensionPeriod:     pubsub.DefaultReceiveSettings.MaxExtensionPeriod,
		MinExtensionPeriod:     pubsub.DefaultReceiveSettings.MinExtensionPeriod,
		MaxOutstandingMessages: 1,
		NumGoroutines:          1,
		Synchronous:            false,
	}
	return subscription{sub, h, s.logger}, nil
}

type subscription struct {
	*pubsub.Subscription
	Handler
	logger *zap.Logger
}

func (s subscription) Start(ctx context.Context) error {
	s.logger.Debug("start receiving", zap.String("subscription", s.ID()))

	return s.Receive(ctx, func(ctx context.Context, message *pubsub.Message) {
		s.logger.Debug("received message", zap.String("subscription", s.ID()))
		err := s.Handle(ctx, message.Data)
		if err != nil {
			s.logger.Error("failed to process message", zap.Error(err), zap.ByteString("payload", message.Data))
			message.Nack()
			return
		}

		message.Ack()
	})
}
