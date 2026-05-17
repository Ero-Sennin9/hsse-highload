package usecases

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

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
	Title       string
	Description string
	Category    string
	Region      string
	Price       int64
}

func (uc *CreateAdUseCase) Execute(ctx context.Context, in CreateAdInput) (*entities.Ad, error) {
	title := strings.TrimSpace(in.Title)
	desc := strings.TrimSpace(in.Description)
	category := strings.TrimSpace(in.Category)
	region := strings.TrimSpace(in.Region)

	switch {
	case utf8.RuneCountInString(title) < 5 || utf8.RuneCountInString(title) > 120:
		return nil, errors.New("title length must be 5-120")
	case utf8.RuneCountInString(desc) < 20 || utf8.RuneCountInString(desc) > 5000:
		return nil, errors.New("description length must be 20-5000")
	case category == "":
		return nil, errors.New("category required")
	case region == "":
		return nil, errors.New("region required")
	case in.Price < 0:
		return nil, errors.New("price must be >= 0")
	}

	now := time.Now().UTC()
	ad := &entities.Ad{
		ID:          uuid.NewString(),
		Title:       title,
		Description: desc,
		Category:    category,
		Region:      region,
		Price:       in.Price,
		Status:      entities.AdStatusModerationPending,
		CreatedAt:   now,
	}
	if err := uc.ads.Save(ctx, ad); err != nil {
		return nil, err
	}
	return ad, nil
}
