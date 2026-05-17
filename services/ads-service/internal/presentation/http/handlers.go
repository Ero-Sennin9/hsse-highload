package httpapi

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"mime"
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
	upload  *usecases.UploadAdPhotoUseCase
	db      *sql.DB
}

func NewRouter(
	create *usecases.CreateAdUseCase,
	publish *usecases.PublishAdUseCase,
	get *usecases.GetAdUseCase,
	upload *usecases.UploadAdPhotoUseCase,
	db *sql.DB,
) http.Handler {
	rt := &Router{create: create, publish: publish, get: get, upload: upload, db: db}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", rt.health)
	mux.HandleFunc("POST /api/v1/ads", rt.createAd)
	mux.HandleFunc("POST /api/v1/ads/{id}/photos", rt.uploadPhoto)
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
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Region      string `json:"region"`
	Price       int64  `json:"price"`
}

type photoResponse struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	ContentType string `json:"contentType"`
	SizeBytes   int64  `json:"sizeBytes"`
	Position    int    `json:"position"`
}

type adResponse struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Category    string          `json:"category"`
	Region      string          `json:"region"`
	Price       int64           `json:"price"`
	Status      string          `json:"status"`
	Photos      []photoResponse `json:"photos"`
	PublishedAt *string         `json:"publishedAt,omitempty"`
}

func (rt *Router) createAd(w http.ResponseWriter, r *http.Request) {
	var body createAdRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ad, err := rt.create.Execute(r.Context(), usecases.CreateAdInput{
		Title:       body.Title,
		Description: body.Description,
		Category:    body.Category,
		Region:      body.Region,
		Price:       body.Price,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toAdResponse(ad))
}

func (rt *Router) uploadPhoto(w http.ResponseWriter, r *http.Request) {
	adID := strings.TrimSpace(r.PathValue("id"))
	if adID == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, usecases.MaxPhotoBytes+512)
	if err := r.ParseMultipartForm(usecases.MaxPhotoBytes); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file field required")
		return
	}
	defer func() { _ = file.Close() }()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" || contentType == "application/octet-stream" {
		if ct := usecases.ContentTypeFromExt(usecases.PhotoExtFromFilename(header.Filename)); ct != "" {
			contentType = ct
		}
	}
	if contentType == "" {
		contentType = mime.TypeByExtension(usecases.PhotoExtFromFilename(header.Filename))
	}

	limited := io.LimitReader(file, usecases.MaxPhotoBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		writeError(w, http.StatusBadRequest, "read file failed")
		return
	}
	if int64(len(data)) > usecases.MaxPhotoBytes {
		writeError(w, http.StatusBadRequest, "file too large")
		return
	}
	if len(data) == 0 {
		writeError(w, http.StatusBadRequest, "empty file")
		return
	}

	photo, err := rt.upload.Execute(r.Context(), usecases.UploadAdPhotoInput{
		AdID:        adID,
		ContentType: contentType,
		Size:        int64(len(data)),
		Body:        bytes.NewReader(data),
	})
	if err != nil {
		switch {
		case errors.Is(err, usecases.ErrAdNotFound):
			writeError(w, http.StatusNotFound, "not found")
		case strings.Contains(err.Error(), "photo limit"):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	writeJSON(w, http.StatusCreated, toPhotoResponse(photo))
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
	full, err := rt.get.Execute(r.Context(), id)
	if err == nil && full != nil {
		ad = full
	}
	writeJSON(w, http.StatusOK, toAdResponse(ad))
}

func toAdResponse(ad *entities.Ad) adResponse {
	out := adResponse{
		ID:          ad.ID,
		Title:       ad.Title,
		Description: ad.Description,
		Category:    ad.Category,
		Region:      ad.Region,
		Price:       ad.Price,
		Status:      string(ad.Status),
		Photos:      make([]photoResponse, 0, len(ad.Photos)),
	}
	for _, p := range ad.Photos {
		out.Photos = append(out.Photos, toPhotoResponse(&p))
	}
	if ad.PublishedAt != nil {
		s := ad.PublishedAt.UTC().Format(time.RFC3339Nano)
		out.PublishedAt = &s
	}
	return out
}

func toPhotoResponse(p *entities.AdPhoto) photoResponse {
	return photoResponse{
		ID:          p.ID,
		URL:         p.URL,
		ContentType: p.ContentType,
		SizeBytes:   p.SizeBytes,
		Position:    p.Position,
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
