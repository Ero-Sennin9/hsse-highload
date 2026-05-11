package ports

import (
	"context"

	"ads-service/internal/domain/entities"
)

type AdsRepository interface {
	Save(ctx context.Context, ad *entities.Ad) error
	GetByID(ctx context.Context, id string) (*entities.Ad, error)
}
