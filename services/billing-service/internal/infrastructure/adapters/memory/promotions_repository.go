package memory

import (
	"context"
	"errors"
	"sync"

	"billing-service/internal/domain/entities"
	"billing-service/internal/domain/ports"
)

var _ ports.PromotionsRepository = (*PromotionsRepository)(nil)

type PromotionsRepository struct {
	mu   sync.Mutex
	rows []*entities.Promotion
}

func NewPromotionsRepository() *PromotionsRepository {
	return &PromotionsRepository{}
}

func (r *PromotionsRepository) Save(_ context.Context, p *entities.Promotion) error {
	if p == nil || p.ID == "" {
		return errors.New("promotion id required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *p
	r.rows = append(r.rows, &cp)
	return nil
}
