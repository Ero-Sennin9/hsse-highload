package httpadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sony/gobreaker"

	"billing-service/internal/domain/ports"
)

var _ ports.AdsCatalog = (*AdsCatalogAdapter)(nil)

type AdsCatalogAdapter struct {
	baseURL string
	client  ports.HTTPClient
	breaker *gobreaker.CircuitBreaker
}

func NewAdsCatalogAdapter(client ports.HTTPClient) (*AdsCatalogAdapter, error) {
	if client == nil {
		return nil, fmt.Errorf("nil http client")
	}
	base, err := resolveAdsBaseURL()
	if err != nil {
		return nil, err
	}
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "ads-catalog",
		MaxRequests: 3,
		Interval:    20 * time.Second,
		Timeout:     12 * time.Second,
		ReadyToTrip: func(c gobreaker.Counts) bool {
			return c.ConsecutiveFailures >= 5
		},
	})
	return &AdsCatalogAdapter{
		baseURL: strings.TrimRight(base, "/"),
		client:  client,
		breaker: cb,
	}, nil
}

func resolveAdsBaseURL() (string, error) {
	if v := strings.TrimSpace(os.Getenv("ADS_SERVICE_BASE_URL")); v != "" {
		return v, nil
	}
	host := strings.TrimSpace(os.Getenv("ADS_SERVICE_HOST"))
	if host == "" {
		return "", fmt.Errorf("set ADS_SERVICE_BASE_URL or ADS_SERVICE_HOST")
	}
	port := strings.TrimSpace(os.Getenv("ADS_SERVICE_PORT"))
	if port == "" {
		return "", fmt.Errorf("set ADS_SERVICE_PORT when using ADS_SERVICE_HOST")
	}
	return fmt.Sprintf("http://%s:%s", host, port), nil
}

type adsAPIResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (a *AdsCatalogAdapter) MustBePublished(ctx context.Context, adID string) error {
	adID = strings.TrimSpace(adID)
	if adID == "" {
		return fmt.Errorf("empty ad id")
	}
	_, err := a.breaker.Execute(func() (any, error) {
		return nil, a.getWithRetry(ctx, adID)
	})
	return err
}

func (a *AdsCatalogAdapter) getWithRetry(ctx context.Context, adID string) error {
	backoff := 40 * time.Millisecond
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			t := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				t.Stop()
				return ctx.Err()
			case <-t.C:
			}
			backoff *= 2
		}
		lastErr = a.getOnce(ctx, adID)
		if lastErr == nil {
			return nil
		}
		if !isRetryableHTTPError(lastErr) {
			return lastErr
		}
	}
	return lastErr
}

func isRetryableHTTPError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "status 502") ||
		strings.Contains(s, "status 503") ||
		strings.Contains(s, "status 504") ||
		strings.Contains(s, "timeout") ||
		strings.Contains(s, "connection reset") ||
		strings.Contains(s, "EOF")
}

func (a *AdsCatalogAdapter) getOnce(ctx context.Context, adID string) error {
	url := fmt.Sprintf("%s/api/v1/ads/%s", a.baseURL, adID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("ads-service: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		var parsed adsAPIResponse
		if err := json.Unmarshal(body, &parsed); err != nil {
			return fmt.Errorf("ads-service: decode: %w", err)
		}
		if parsed.Status != "published" {
			return fmt.Errorf("ad %s is not published", adID)
		}
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("ads-service: ad not found")
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return fmt.Errorf("ads-service: GET /api/v1/ads/{id}: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	default:
		return fmt.Errorf("ads-service: GET /api/v1/ads/{id}: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}
