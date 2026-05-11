package usecases

import (
	"context"
	"testing"

	"billing-service/internal/domain/entities"
	"billing-service/internal/infrastructure/adapters/memory"
)

type fakeCatalog struct {
	err error
}

func (f fakeCatalog) MustBePublished(context.Context, string) error {
	return f.err
}

func TestPurchasePromotionUseCase_Execute_CatalogBlocks(t *testing.T) {
	repo := memory.NewPromotionsRepository()
	uc, err := NewPurchasePromotionUseCase(fakeCatalog{err: errNotPublished}, repo)
	if err != nil {
		t.Fatal(err)
	}
	_, err = uc.Execute(context.Background(), PurchasePromotionInput{AdID: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

var errNotPublished = errFake("not published")

type errFake string

func (e errFake) Error() string { return string(e) }

func TestPurchasePromotionUseCase_Execute_OK(t *testing.T) {
	repo := memory.NewPromotionsRepository()
	uc, err := NewPurchasePromotionUseCase(fakeCatalog{}, repo)
	if err != nil {
		t.Fatal(err)
	}
	p, err := uc.Execute(context.Background(), PurchasePromotionInput{AdID: "ad-1"})
	if err != nil {
		t.Fatal(err)
	}
	if p.AdID != "ad-1" || p.Status != entities.PromotionStatusActive {
		t.Fatalf("unexpected promotion: %+v", p)
	}
}
