package pubsub

import (
	"cloud.google.com/go/pubsub"
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
)

type PublisherCreator interface {
	Create(context.Context, string) (Publisher, error)
}

type Publisher interface {
	Publish(context.Context, any) error
}

type publishFactory struct {
	cli    *pubsub.Client
	logger *zap.Logger
}

func NewPublisherFactory(cli *pubsub.Client, logger *zap.Logger) PublisherCreator {
	return &publishFactory{cli, logger}
}

func (p publishFactory) Create(ctx context.Context, topic string) (Publisher, error) {
	t := p.cli.Topic(topic)
	ok, err := t.Exists(ctx)
	if err != nil {
		return nil, err
	}
	p.logger.Debug("check topic existing", zap.Bool("exists", ok), zap.String("topic", topic))

	if !ok {
		return nil, fmt.Errorf("topic does not exist: %s", topic)
	}

	return publisher{t, p.logger}, nil
}

type publisher struct {
	*pubsub.Topic
	logger *zap.Logger
}

func (p publisher) Publish(ctx context.Context, msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	p.logger.Debug("publishing", zap.String("topic", p.ID()), zap.ByteString("payload", data))
	_ = p.Topic.Publish(ctx, &pubsub.Message{
		Data: data,
	})

	return nil
}
