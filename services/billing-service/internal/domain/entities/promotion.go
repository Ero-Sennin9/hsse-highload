package entities

import "time"

type PromotionStatus string

const (
	PromotionStatusPending PromotionStatus = "pending_publication"
	PromotionStatusActive  PromotionStatus = "active"
)

type Promotion struct {
	ID        string
	AdID      string
	Status    PromotionStatus
	CreatedAt time.Time
}
