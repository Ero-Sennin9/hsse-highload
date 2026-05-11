package usecases

import (
	"context"
	"errors"

	"ads-service/internal/domain/entities"
	"ads-service/internal/domain/ports"
)

type GetAdUseCase struct {
	ads ports.AdsRepository
}

func NewGetAdUseCase(ads ports.AdsRepository) (*GetAdUseCase, error) {
	if ads == nil {
		return nil, errors.New("nil ads repository")
	}
	return &GetAdUseCase{ads: ads}, nil
}

func (uc *GetAdUseCase) Execute(ctx context.Context, id string) (*entities.Ad, error) {
	if id == "" {
		return nil, errors.New("empty id")
	}
	ad, err := uc.ads.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ad == nil {
		return nil, ErrAdNotFound
	}
	return ad, nil
}
