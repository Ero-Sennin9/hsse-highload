package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"billing-service/internal/domain/entities"
	"billing-service/internal/domain/ports"
)

type PurchasePromotionUseCase struct {
	catalog ports.AdsCatalog
	store   ports.PromotionsRepository
}

func NewPurchasePromotionUseCase(catalog ports.AdsCatalog, store ports.PromotionsRepository) (*PurchasePromotionUseCase, error) {
	if catalog == nil {
		return nil, errors.New("nil ads catalog")
	}
	if store == nil {
		return nil, errors.New("nil promotions repository")
	}
	return &PurchasePromotionUseCase{catalog: catalog, store: store}, nil
}

type PurchasePromotionInput struct {
	AdID string
}

func (uc *PurchasePromotionUseCase) Execute(ctx context.Context, in PurchasePromotionInput) (*entities.Promotion, error) {
	if err := uc.catalog.MustBePublished(ctx, in.AdID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	p := &entities.Promotion{
		ID:        uuid.NewString(),
		AdID:      in.AdID,
		Status:    entities.PromotionStatusActive,
		CreatedAt: now,
	}
	if err := uc.store.Save(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}
