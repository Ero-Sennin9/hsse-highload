package httpapi

import (
	"net/http"

	"golang.org/x/time/rate"
)

func RateLimitCreate(inner http.Handler, lim *rate.Limiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/ads" {
			if !lim.Allow() {
				writeError(w, http.StatusTooManyRequests, "rate limited")
				return
			}
		}
		inner.ServeHTTP(w, r)
	})
}
