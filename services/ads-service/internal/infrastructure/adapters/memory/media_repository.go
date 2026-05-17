package memory

import (
	"context"
	"errors"
	"sort"
	"sync"

	"ads-service/internal/domain/entities"
	"ads-service/internal/domain/ports"
)

var _ ports.MediaRepository = (*MediaRepository)(nil)

type MediaRepository struct {
	mu    sync.RWMutex
	rows  []entities.AdPhoto
}

func NewMediaRepository() *MediaRepository {
	return &MediaRepository{}
}

func (r *MediaRepository) Add(_ context.Context, photo *entities.AdPhoto) error {
	if photo == nil || photo.ID == "" {
		return errors.New("invalid photo")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *photo
	r.rows = append(r.rows, cp)
	return nil
}

func (r *MediaRepository) ListByAdID(_ context.Context, adID string) ([]entities.AdPhoto, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []entities.AdPhoto
	for _, p := range r.rows {
		if p.AdID == adID {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Position < out[j].Position })
	return out, nil
}

func (r *MediaRepository) CountByAdID(_ context.Context, adID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n := 0
	for _, p := range r.rows {
		if p.AdID == adID {
			n++
		}
	}
	return n, nil
}
