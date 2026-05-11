package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"ads-service/internal/application/usecases"
	"ads-service/internal/domain/entities"
)

type Router struct {
	create  *usecases.CreateAdUseCase
	publish *usecases.PublishAdUseCase
	get     *usecases.GetAdUseCase
	db      *sql.DB
}

func NewRouter(
	create *usecases.CreateAdUseCase,
	publish *usecases.PublishAdUseCase,
	get *usecases.GetAdUseCase,
	db *sql.DB,
) http.Handler {
	rt := &Router{create: create, publish: publish, get: get, db: db}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", rt.health)
	mux.HandleFunc("POST /api/v1/ads", rt.createAd)
	mux.HandleFunc("POST /api/v1/ads/{id}/publish", rt.publishAd)
	mux.HandleFunc("GET /api/v1/ads/{id}", rt.getAd)
	lim := rate.NewLimiter(rate.Limit(120), 40)
	return RateLimitCreate(mux, lim)
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

type createAdRequest struct {
	Title string `json:"title"`
}

type adResponse struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Status      string  `json:"status"`
	PublishedAt *string `json:"publishedAt,omitempty"`
}

func (rt *Router) createAd(w http.ResponseWriter, r *http.Request) {
	var body createAdRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ad, err := rt.create.Execute(r.Context(), usecases.CreateAdInput{Title: body.Title})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toAdResponse(ad))
}

func (rt *Router) getAd(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	ad, err := rt.get.Execute(r.Context(), id)
	if err != nil {
		if errors.Is(err, usecases.ErrAdNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toAdResponse(ad))
}

func (rt *Router) publishAd(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	ad, err := rt.publish.Execute(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, usecases.ErrAdNotFound):
			writeError(w, http.StatusNotFound, "not found")
		default:
			writeError(w, http.StatusConflict, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusOK, toAdResponse(ad))
}

func toAdResponse(ad *entities.Ad) adResponse {
	out := adResponse{
		ID:     ad.ID,
		Title:  ad.Title,
		Status: string(ad.Status),
	}
	if ad.PublishedAt != nil {
		s := ad.PublishedAt.UTC().Format(time.RFC3339Nano)
		out.PublishedAt = &s
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
