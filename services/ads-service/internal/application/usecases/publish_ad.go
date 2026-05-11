package usecases

import (
	"context"
	"errors"
	"time"

	"ads-service/internal/domain/entities"
	"ads-service/internal/domain/ports"
)

type PublishAdUseCase struct {
	ads ports.AdsRepository
}

func NewPublishAdUseCase(ads ports.AdsRepository) (*PublishAdUseCase, error) {
	if ads == nil {
		return nil, errors.New("nil ads repository")
	}
	return &PublishAdUseCase{ads: ads}, nil
}

func (uc *PublishAdUseCase) Execute(ctx context.Context, adID string) (*entities.Ad, error) {
	if adID == "" {
		return nil, errors.New("ad id required")
	}
	ad, err := uc.ads.GetByID(ctx, adID)
	if err != nil {
		return nil, err
	}
	if ad == nil {
		return nil, ErrAdNotFound
	}
	if ad.Status != entities.AdStatusModerationPending {
		return nil, errors.New("only moderation_pending can be published")
	}
	now := time.Now().UTC()
	ad.Status = entities.AdStatusPublished
	ad.PublishedAt = &now
	if err := uc.ads.Save(ctx, ad); err != nil {
		return nil, err
	}
	return ad, nil
}
