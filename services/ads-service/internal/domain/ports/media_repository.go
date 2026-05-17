package ports

import (
	"context"

	"ads-service/internal/domain/entities"
)

type MediaRepository interface {
	Add(ctx context.Context, photo *entities.AdPhoto) error
	ListByAdID(ctx context.Context, adID string) ([]entities.AdPhoto, error)
	CountByAdID(ctx context.Context, adID string) (int, error)
}
