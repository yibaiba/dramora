package media

import (
	"context"
	"io"
)

type Object struct {
	URI         string
	ContentType string
	SizeBytes   int64
}

type Storage interface {
	Put(ctx context.Context, key string, body io.Reader, contentType string) (Object, error)
	Get(ctx context.Context, uri string) (io.ReadCloser, error)
}
