package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"ads-service/internal/domain/entities"
	"ads-service/internal/domain/ports"
)

var _ ports.AdsRepository = (*AdsRepository)(nil)

type AdsRepository struct {
	db *sql.DB
}

func NewAdsRepository(db *sql.DB) (*AdsRepository, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	return &AdsRepository{db: db}, nil
}

func (r *AdsRepository) Save(ctx context.Context, ad *entities.Ad) error {
	if ad == nil || ad.ID == "" {
		return errors.New("ad id required")
	}
	var pub sql.NullTime
	if ad.PublishedAt != nil {
		pub = sql.NullTime{Time: *ad.PublishedAt, Valid: true}
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO ads (id, title, description, category, region, price, status, created_at, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			region = EXCLUDED.region,
			price = EXCLUDED.price,
			status = EXCLUDED.status,
			published_at = EXCLUDED.published_at`,
		ad.ID, ad.Title, ad.Description, ad.Category, ad.Region, ad.Price,
		string(ad.Status), ad.CreatedAt.UTC(), pub,
	)
	return err
}

func (r *AdsRepository) GetByID(ctx context.Context, id string) (*entities.Ad, error) {
	if id == "" {
		return nil, errors.New("empty id")
	}
	row := r.db.QueryRowContext(ctx, `
		SELECT id, title, description, category, region, price, status, created_at, published_at
		FROM ads WHERE id = $1`, id,
	)
	var (
		ad          entities.Ad
		status      string
		createdAt   time.Time
		publishedAt sql.NullTime
	)
	if err := row.Scan(
		&ad.ID, &ad.Title, &ad.Description, &ad.Category, &ad.Region, &ad.Price,
		&status, &createdAt, &publishedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	ad.Status = entities.AdStatus(status)
	ad.CreatedAt = createdAt.UTC()
	if publishedAt.Valid {
		t := publishedAt.Time.UTC()
		ad.PublishedAt = &t
	}
	return &ad, nil
}
