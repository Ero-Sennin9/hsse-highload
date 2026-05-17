package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"ads-service/internal/domain/entities"
	"ads-service/internal/domain/ports"
)

var _ ports.MediaRepository = (*MediaRepository)(nil)

type MediaRepository struct {
	db *sql.DB
}

func NewMediaRepository(db *sql.DB) (*MediaRepository, error) {
	if db == nil {
		return nil, errors.New("nil db")
	}
	return &MediaRepository{db: db}, nil
}

func (r *MediaRepository) Add(ctx context.Context, photo *entities.AdPhoto) error {
	if photo == nil || photo.ID == "" || photo.AdID == "" {
		return errors.New("invalid photo")
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO ad_media (id, ad_id, storage_key, content_type, size_bytes, position, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		photo.ID, photo.AdID, photo.StorageKey, photo.ContentType, photo.SizeBytes,
		photo.Position, photo.CreatedAt.UTC(),
	)
	return err
}

func (r *MediaRepository) ListByAdID(ctx context.Context, adID string) ([]entities.AdPhoto, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, ad_id, storage_key, content_type, size_bytes, position, created_at
		FROM ad_media WHERE ad_id = $1 ORDER BY position`, adID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []entities.AdPhoto
	for rows.Next() {
		var p entities.AdPhoto
		var createdAt time.Time
		if err := rows.Scan(
			&p.ID, &p.AdID, &p.StorageKey, &p.ContentType, &p.SizeBytes, &p.Position, &createdAt,
		); err != nil {
			return nil, err
		}
		p.CreatedAt = createdAt.UTC()
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *MediaRepository) CountByAdID(ctx context.Context, adID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM ad_media WHERE ad_id = $1`, adID).Scan(&n)
	return n, err
}
