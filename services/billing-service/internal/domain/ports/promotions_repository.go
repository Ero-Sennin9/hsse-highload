package ports

import (
	"context"

	"billing-service/internal/domain/entities"
)

type PromotionsRepository interface {
	Save(ctx context.Context, p *entities.Promotion) error
}
