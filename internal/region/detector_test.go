package region

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDetectorCachesCountryCode(t *testing.T) {
	t.Parallel()

	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"country_code":"cn"}`)); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	detector := NewDetector(
		WithEndpoint(server.URL),
		WithHTTPClient(server.Client()),
	)

	code, err := detector.CountryCode(context.Background())
	if err != nil {
		t.Fatalf("CountryCode error: %v", err)
	}
	if code != "CN" {
		t.Fatalf("expected CN, got %s", code)
	}

	code, err = detector.CountryCode(context.Background())
	if err != nil {
		t.Fatalf("CountryCode second call error: %v", err)
	}
	if code != "CN" {
		t.Fatalf("expected cached CN, got %s", code)
	}

	if hits != 1 {
		t.Fatalf("expected 1 upstream hit, got %d", hits)
	}
}

func TestDetectorReturnsErrorOnFailure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	detector := NewDetector(
		WithEndpoint(server.URL),
		WithHTTPClient(server.Client()),
	)

	if _, err := detector.CountryCode(context.Background()); err == nil {
		t.Fatal("expected error for non-200 status")
	}
}
