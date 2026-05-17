package entities

import "time"

type AdStatus string

const (
	AdStatusDraft             AdStatus = "draft"
	AdStatusModerationPending AdStatus = "moderation_pending"
	AdStatusPublished         AdStatus = "published"
	AdStatusRejected          AdStatus = "rejected"
)

const MaxPhotosPerAd = 8

type Ad struct {
	ID          string
	Title       string
	Description string
	Category    string
	Region      string
	Price       int64
	Status      AdStatus
	Photos      []AdPhoto
	CreatedAt   time.Time
	PublishedAt *time.Time
}

type AdPhoto struct {
	ID          string
	AdID        string
	StorageKey  string
	ContentType string
	SizeBytes   int64
	Position    int
	URL         string
	CreatedAt   time.Time
}
