package postgres

import (
	"context"
	"database/sql"
	"errors"

	"billing-service/internal/domain/entities"
	"billing-service/internal/domain/ports"
)

var _ ports.PromotionsRepository = (*PromotionsRepository)(nil)

type PromotionsRepository struct {
	db *sql.DB
}

func NewPromotionsRepository(db *sql.DB) (*PromotionsRepository, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	return &PromotionsRepository{db: db}, nil
}

func (r *PromotionsRepository) Save(ctx context.Context, p *entities.Promotion) error {
	if p == nil || p.ID == "" {
		return errors.New("promotion id required")
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO promotions (id, ad_id, status, created_at)
		VALUES ($1, $2, $3, $4)`,
		p.ID, p.AdID, string(p.Status), p.CreatedAt.UTC(),
	)
	return err
}
