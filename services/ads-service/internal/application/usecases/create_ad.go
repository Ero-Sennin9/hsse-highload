package usecases

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"ads-service/internal/domain/entities"
	"ads-service/internal/domain/ports"
)

type CreateAdUseCase struct {
	ads ports.AdsRepository
}

func NewCreateAdUseCase(ads ports.AdsRepository) (*CreateAdUseCase, error) {
	if ads == nil {
		return nil, errors.New("nil ads repository")
	}
	return &CreateAdUseCase{ads: ads}, nil
}

type CreateAdInput struct {
	Title string
}

func (uc *CreateAdUseCase) Execute(ctx context.Context, in CreateAdInput) (*entities.Ad, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, errors.New("title required")
	}
	now := time.Now().UTC()
	ad := &entities.Ad{
		ID:        uuid.NewString(),
		Title:     title,
		Status:    entities.AdStatusModerationPending,
		CreatedAt: now,
	}
	if err := uc.ads.Save(ctx, ad); err != nil {
		return nil, err
	}
	return ad, nil
}
