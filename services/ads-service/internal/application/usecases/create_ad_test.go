package usecases

import (
	"context"
	"strings"
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
	desc := strings.Repeat("x", 25)
	ad, err := uc.Execute(context.Background(), CreateAdInput{
		Title:       "  bike pro  ",
		Description: desc,
		Category:    "transport",
		Region:      "moscow",
		Price:       15000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ad.Title != "bike pro" {
		t.Fatalf("title: got %q", ad.Title)
	}
	if ad.Description != desc {
		t.Fatalf("description mismatch")
	}
	if ad.Status != entities.AdStatusModerationPending {
		t.Fatalf("status: got %q", ad.Status)
	}
	if ad.ID == "" {
		t.Fatal("expected id")
	}
}
