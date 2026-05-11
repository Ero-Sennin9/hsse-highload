package memory

import (
	"context"
	"errors"
	"sync"

	"ads-service/internal/domain/entities"
	"ads-service/internal/domain/ports"
)

var _ ports.AdsRepository = (*AdsRepository)(nil)

type AdsRepository struct {
	mu   sync.RWMutex
	rows map[string]*entities.Ad
}

func NewAdsRepository() *AdsRepository {
	return &AdsRepository{rows: make(map[string]*entities.Ad)}
}

func (r *AdsRepository) Save(_ context.Context, ad *entities.Ad) error {
	if ad == nil || ad.ID == "" {
		return errors.New("ad id required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *ad
	r.rows[ad.ID] = &cp
	return nil
}

func (r *AdsRepository) GetByID(_ context.Context, id string) (*entities.Ad, error) {
	if id == "" {
		return nil, errors.New("empty id")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	ad, ok := r.rows[id]
	if !ok {
		return nil, nil
	}
	cp := *ad
	return &cp, nil
}
