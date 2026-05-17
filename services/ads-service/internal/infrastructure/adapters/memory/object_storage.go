package memory

import (
	"bytes"
	"context"
	"io"
	"sync"

	"ads-service/internal/domain/ports"
)

var _ ports.ObjectStorage = (*ObjectStorage)(nil)

type ObjectStorage struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewObjectStorage() *ObjectStorage {
	return &ObjectStorage{data: make(map[string][]byte)}
}

func (s *ObjectStorage) Put(_ context.Context, key string, body io.Reader, size int64, _ string) error {
	buf, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.data[key] = buf
	s.mu.Unlock()
	_ = size
	return nil
}

func (s *ObjectStorage) PublicURL(key string) string {
	return "memory://" + key
}

func (s *ObjectStorage) Get(key string) []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return bytes.Clone(s.data[key])
}
