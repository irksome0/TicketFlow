package storage

import (
	"context"
	"io"
)

type Provider string

const (
	ProviderLocal Provider = "local"
	ProviderR2    Provider = "r2"
)

// FileStorage describes a private attachment storage provider.
type FileStorage interface {
	Save(ctx context.Context, key string, file io.Reader, contentType string) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}
