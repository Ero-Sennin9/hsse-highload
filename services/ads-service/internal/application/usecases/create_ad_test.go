package usecases

import (
	"context"
	"testing"

	"ads-service/internal/domain/entities"
	"ads-service/internal/infrastructure/adapters/memory"
)

func TestCreateAdUseCase_Execute(t *testing.T) {
	repo := memory.NewAdsRepository()
	uc, err := NewCreateAdUseCase(repo)
	if err != nil {
		t.Fatal(err)
	}
	ad, err := uc.Execute(context.Background(), CreateAdInput{Title: "  bike  "})
	if err != nil {
		t.Fatal(err)
	}
	if ad.Title != "bike" {
		t.Fatalf("title: got %q", ad.Title)
	}
	if ad.Status != entities.AdStatusModerationPending {
		t.Fatalf("status: got %q", ad.Status)
	}
	if ad.ID == "" {
		t.Fatal("expected id")
	}
}
