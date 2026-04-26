package jobs

import "context"

type Client interface {
	Enqueue(ctx context.Context, job Job) error
}

type NoopClient struct{}

func NewNoopClient() *NoopClient {
	return &NoopClient{}
}

func (c *NoopClient) Enqueue(_ context.Context, _ Job) error {
	return nil
}
