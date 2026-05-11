package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"billing-service/internal/application/usecases"
	"billing-service/internal/domain/entities"
)

type Router struct {
	purchase *usecases.PurchasePromotionUseCase
	db       *sql.DB
}

func NewRouter(purchase *usecases.PurchasePromotionUseCase, db *sql.DB) http.Handler {
	rt := &Router{purchase: purchase, db: db}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", rt.health)
	mux.HandleFunc("POST /api/v1/ads/{id}/promotions", rt.purchasePromotion)
	return mux
}

func (rt *Router) health(w http.ResponseWriter, r *http.Request) {
	if rt.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := rt.db.PingContext(ctx); err != nil {
			writeError(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

type promotionResponse struct {
	ID        string `json:"id"`
	AdID      string `json:"adId"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

func (rt *Router) purchasePromotion(w http.ResponseWriter, r *http.Request) {
	adID := strings.TrimSpace(r.PathValue("id"))
	if adID == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	p, err := rt.purchase.Execute(r.Context(), usecases.PurchasePromotionInput{AdID: adID})
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "not found"):
			writeError(w, http.StatusNotFound, msg)
		case strings.Contains(msg, "not published"):
			writeError(w, http.StatusConflict, msg)
		default:
			writeError(w, http.StatusBadGateway, msg)
		}
		return
	}
	writeJSON(w, http.StatusCreated, toPromotionResponse(p))
}

func toPromotionResponse(p *entities.Promotion) promotionResponse {
	return promotionResponse{
		ID:        p.ID,
		AdID:      p.AdID,
		Status:    string(p.Status),
		CreatedAt: p.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
