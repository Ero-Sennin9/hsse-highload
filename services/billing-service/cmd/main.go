package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"billing-service/internal/di"
)

func main() {
	addr := getenv("HTTP_ADDR", ":8082")
	handler, err := di.InitializeHTTPHandler()
	if err != nil {
		log.Fatalf("di: %v", err)
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("billing-service listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
