package main

import (
	"context"
	"net/http"
	"os"
	"time"
)

const defaultHealthcheckURL = "http://127.0.0.1:8080/api/v1/health/live"

// runHealthcheck lets minimal container images probe the API without shipping
// a shell, curl, or wget. A zero exit status means the liveness endpoint is OK.
func runHealthcheck() int {
	url := os.Getenv("DAFTAR_HEALTHCHECK_URL")
	if url == "" {
		url = defaultHealthcheckURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 1
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 1
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
