package ports

import (
	"context"
	"io"
)

type ObjectStorage interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	PublicURL(key string) string
}
