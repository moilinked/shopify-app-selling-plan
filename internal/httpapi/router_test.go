package httpapi

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestAdminRouteRequiresJWT(t *testing.T) {
	cfg := config.Config{
		Port:             "9998",
		ShopifyAPIKey:    "test-key",
		ShopifyAPISecret: "test-secret",
	}
	srv := httptest.NewServer(NewRouter(cfg))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/admin/ping")
	if err != nil {
		t.Fatalf("GET /admin/ping: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestAdminRouteAcceptsValidJWTAndLogsClaims(t *testing.T) {
	const apiKey = "test-key"
	const apiSecret = "test-secret"

	cfg := config.Config{
		Port:             "9998",
		ShopifyAPIKey:    apiKey,
		ShopifyAPISecret: apiSecret,
		DebugAuth:        true,
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
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/admin/ping", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /admin/ping: %v", err)
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

func makeSignedAppProxyQuery(secret string) string {
	values := url.Values{}
	values.Set("shop", "devloop-4.myshopify.com")
	values.Set("logged_in_customer_id", "89466365215417")
	values.Set("path_prefix", "/apps/my-app-test")
	values.Set("timestamp", "1772420951")

	signature := computeSHA256HMACHex(canonicalizeProxyParams(values), secret)
	values.Set("signature", signature)
	return values.Encode()
}

func TestAppProxyRouteRequiresValidSignature(t *testing.T) {
	const apiSecret = "test-secret"

	cfg := config.Config{
		Port:             "9998",
		ShopifyAPIKey:    "test-key",
		ShopifyAPISecret: apiSecret,
	}
	srv := httptest.NewServer(NewRouter(cfg))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/app/ping")
	if err != nil {
		t.Fatalf("GET /app/ping: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	query := makeSignedAppProxyQuery(apiSecret)
	resp2, err := http.Get(srv.URL + "/app/ping?" + query)
	if err != nil {
		t.Fatalf("GET /app/ping signed: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp2.Body)
		t.Fatalf("signed status: got %d, want %d; body=%s", resp2.StatusCode, http.StatusOK, string(body))
	}

	badQuery := strings.Replace(query, "signature=", "signature=bad", 1)
	resp3, err := http.Get(srv.URL + "/app/ping?" + badQuery)
	if err != nil {
		t.Fatalf("GET /app/ping bad signature: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad signature status: got %d, want %d", resp3.StatusCode, http.StatusUnauthorized)
	}
}
