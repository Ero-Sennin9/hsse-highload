package usecases

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"ads-service/internal/domain/entities"
	"ads-service/internal/domain/ports"
)

const MaxPhotoBytes = 5 * 1024 * 1024

var allowedPhotoTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type UploadAdPhotoInput struct {
	AdID        string
	ContentType string
	Size        int64
	Body        io.Reader
}

type UploadAdPhotoUseCase struct {
	ads     ports.AdsRepository
	media   ports.MediaRepository
	storage ports.ObjectStorage
}

func NewUploadAdPhotoUseCase(
	ads ports.AdsRepository,
	media ports.MediaRepository,
	storage ports.ObjectStorage,
) (*UploadAdPhotoUseCase, error) {
	if ads == nil || media == nil || storage == nil {
		return nil, errors.New("nil dependency")
	}
	return &UploadAdPhotoUseCase{ads: ads, media: media, storage: storage}, nil
}

func (uc *UploadAdPhotoUseCase) Execute(ctx context.Context, in UploadAdPhotoInput) (*entities.AdPhoto, error) {
	adID := strings.TrimSpace(in.AdID)
	if adID == "" {
		return nil, errors.New("ad id required")
	}
	if in.Size <= 0 || in.Size > MaxPhotoBytes {
		return nil, fmt.Errorf("photo size must be 1-%d bytes", MaxPhotoBytes)
	}
	ext, ok := allowedPhotoTypes[strings.ToLower(strings.TrimSpace(in.ContentType))]
	if !ok {
		return nil, errors.New("unsupported content type (jpeg, png, webp)")
	}
	if in.Body == nil {
		return nil, errors.New("empty body")
	}

	ad, err := uc.ads.GetByID(ctx, adID)
	if err != nil {
		return nil, err
	}
	if ad == nil {
		return nil, ErrAdNotFound
	}

	count, err := uc.media.CountByAdID(ctx, adID)
	if err != nil {
		return nil, err
	}
	if count >= entities.MaxPhotosPerAd {
		return nil, errors.New("photo limit reached (max 8)")
	}

	photoID := uuid.NewString()
	key := fmt.Sprintf("ads/%s/%s%s", adID, photoID, ext)
	if err := uc.storage.Put(ctx, key, in.Body, in.Size, in.ContentType); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	photo := &entities.AdPhoto{
		ID:          photoID,
		AdID:        adID,
		StorageKey:  key,
		ContentType: in.ContentType,
		SizeBytes:   in.Size,
		Position:    count,
		URL:         uc.storage.PublicURL(key),
		CreatedAt:   now,
	}
	if err := uc.media.Add(ctx, photo); err != nil {
		return nil, err
	}
	return photo, nil
}

func PhotoExtFromFilename(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg":
		return ".jpg"
	case ".png":
		return ".png"
	case ".webp":
		return ".webp"
	default:
		return ""
	}
}

func ContentTypeFromExt(ext string) string {
	switch ext {
	case ".jpg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	default:
		return ""
	}
}
