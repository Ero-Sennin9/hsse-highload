package entities

import "time"

type AdStatus string

const (
	AdStatusDraft               AdStatus = "draft"
	AdStatusModerationPending   AdStatus = "moderation_pending"
	AdStatusPublished           AdStatus = "published"
	AdStatusRejected            AdStatus = "rejected"
)

type Ad struct {
	ID          string
	Title       string
	Status      AdStatus
	CreatedAt   time.Time
	PublishedAt *time.Time
}
