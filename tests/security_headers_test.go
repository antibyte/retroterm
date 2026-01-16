package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"strings"
	"github.com/antibyte/retroterm/pkg/middleware"
)

func TestSecurityHeaders(t *testing.T) {
	// Create a mock handler that we'll wrap
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap the handler with our security middleware
	secureHandler := middleware.WithSecurityHeaders(mockHandler)

	// Create a test request
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Record the response
	rr := httptest.NewRecorder()
	secureHandler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Verify required security headers
	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "SAMEORIGIN",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Content-Security-Policy": "default-src 'self'",
	}

	for header, expected := range headers {
		val := rr.Header().Get(header)
		if val == "" {
			t.Errorf("Header %s is missing", header)
		}
		if !strings.Contains(val, expected) {
			t.Errorf("Header %s does not contain expected value. Got: %s, Want to contain: %s",
				header, val, expected)
		}
	}
}
