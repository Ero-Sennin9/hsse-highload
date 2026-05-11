package ports

import "context"

type AdsCatalog interface {
	MustBePublished(ctx context.Context, adID string) error
}
