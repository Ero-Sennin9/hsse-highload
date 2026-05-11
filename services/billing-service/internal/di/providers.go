package di

import (
	"net/http"
	"time"
)

func provideHTTPClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Second}
}
