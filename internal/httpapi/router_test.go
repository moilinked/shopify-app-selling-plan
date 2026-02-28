package httpapi

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"shopify-app-authentication/internal/config"

	"github.com/golang-jwt/jwt/v5"
)

func TestPing(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Port:             "9998",
		ShopifyAPIKey:    "test-key",
		ShopifyAPISecret: "test-secret",
	}
	srv := httptest.NewServer(NewRouter(cfg))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/ping")
	if err != nil {
		t.Fatalf("GET /ping: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if string(b) != "pong" {
		t.Fatalf("body: got %q, want %q", string(b), "pong")
	}
}

func TestCORSPreflight(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Port:             "9998",
		ShopifyAPIKey:    "test-key",
		ShopifyAPISecret: "test-secret",
	}
	srv := httptest.NewServer(NewRouter(cfg))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodOptions, srv.URL+"/ping", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("OPTIONS /ping: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin: got %q, want %q", got, "*")
	}
	if got := resp.Header.Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatalf("Access-Control-Allow-Headers should not be empty")
	}
}

func makeValidShopifySessionToken(t *testing.T, apiKey, apiSecret string) string {
	t.Helper()

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":  "https://test-shop.myshopify.com/admin",
		"dest": "https://test-shop.myshopify.com",
		"aud":  apiKey,
		"sub":  "42",
		"exp":  now.Add(2 * time.Minute).Unix(),
		"nbf":  now.Add(-1 * time.Minute).Unix(),
		"iat":  now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(apiSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func TestProtectedRouteRequiresJWT(t *testing.T) {
	cfg := config.Config{
		Port:             "9998",
		ShopifyAPIKey:    "test-key",
		ShopifyAPISecret: "test-secret",
	}
	srv := httptest.NewServer(NewRouter(cfg))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/protected/ping")
	if err != nil {
		t.Fatalf("GET /protected/ping: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestProtectedRouteAcceptsValidJWTAndLogsClaims(t *testing.T) {
	const apiKey = "test-key"
	const apiSecret = "test-secret"

	cfg := config.Config{
		Port:             "9998",
		ShopifyAPIKey:    apiKey,
		ShopifyAPISecret: apiSecret,
	}

	var buf bytes.Buffer
	originalWriter := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() {
		log.SetOutput(originalWriter)
	})

	srv := httptest.NewServer(NewRouter(cfg))
	t.Cleanup(srv.Close)

	token := makeValidShopifySessionToken(t, apiKey, apiSecret)
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/protected/ping", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /protected/ping: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status: got %d, want %d; body=%s", resp.StatusCode, http.StatusOK, string(body))
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "shopify_session_token claims=") {
		t.Fatalf("expected claims log, got %q", logOutput)
	}
}
