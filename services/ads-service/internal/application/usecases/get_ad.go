package usecases

import (
	"context"
	"errors"

	"ads-service/internal/domain/entities"
	"ads-service/internal/domain/ports"
)

type GetAdUseCase struct {
	ads     ports.AdsRepository
	media   ports.MediaRepository
	storage ports.ObjectStorage
}

func NewGetAdUseCase(
	ads ports.AdsRepository,
	media ports.MediaRepository,
	storage ports.ObjectStorage,
) (*GetAdUseCase, error) {
	if ads == nil || media == nil || storage == nil {
		return nil, errors.New("nil dependency")
	}
	return &GetAdUseCase{ads: ads, media: media, storage: storage}, nil
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
	photos, err := uc.media.ListByAdID(ctx, id)
	if err != nil {
		return nil, err
	}
	for i := range photos {
		photos[i].URL = uc.storage.PublicURL(photos[i].StorageKey)
	}
	ad.Photos = photos
	return ad, nil
}
