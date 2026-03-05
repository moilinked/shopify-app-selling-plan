package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"shopify-app-authentication/internal/config"
	"shopify-app-authentication/internal/logger"
	mw "shopify-app-authentication/internal/middleware"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
)

func newTestServer(t *testing.T, cfg config.Config) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(NewRouter(cfg, nil))
	t.Cleanup(srv.Close)
	return srv
}

func defaultTestConfig() config.Config {
	return config.Config{
		Port:             "9998",
		ShopifyAPIKey:    "test-key",
		ShopifyAPISecret: "test-secret",
	}
}

func getJSON(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshal body %q: %v", string(body), err)
	}
	return result
}

func TestPing(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, defaultTestConfig())

	resp, err := http.Get(srv.URL + "/ping")
	if err != nil {
		t.Fatalf("GET /ping: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}

	result := getJSON(t, resp)
	if result["ok"] != true {
		t.Fatalf("ok: got %v, want true", result["ok"])
	}
}

func TestPingWithShopHeader(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, defaultTestConfig())

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/ping", nil)
	req.Header.Set("X-Shop-Domain", "https://my-shop.myshopify.com")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /ping: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}

	result := getJSON(t, resp)
	shop, _ := result["shop"].(string)
	if shop != "my-shop.myshopify.com" {
		t.Fatalf("shop: got %q, want %q", shop, "my-shop.myshopify.com")
	}
}

func TestPingWithoutShopHeader(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, defaultTestConfig())

	resp, err := http.Get(srv.URL + "/ping")
	if err != nil {
		t.Fatalf("GET /ping: %v", err)
	}
	defer resp.Body.Close()

	result := getJSON(t, resp)
	shop, _ := result["shop"].(string)
	if shop != "" {
		t.Fatalf("shop: got %q, want empty", shop)
	}
}

func TestCORSPreflight(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t, defaultTestConfig())

	req, err := http.NewRequest(http.MethodOptions, srv.URL+"/ping", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")

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

func makeValidShopifySessionToken(t *testing.T, apiKey, apiSecret, shopDomain string) string {
	t.Helper()

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":  "https://" + shopDomain + "/admin",
		"dest": "https://" + shopDomain,
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
	srv := newTestServer(t, defaultTestConfig())

	resp, err := http.Get(srv.URL + "/admin/ping")
	if err != nil {
		t.Fatalf("GET /admin/ping: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestAdminPingReturnsShopFromJWT(t *testing.T) {
	const apiKey = "test-key"
	const apiSecret = "test-secret"
	const shopDomain = "test-shop.myshopify.com"

	cfg := config.Config{
		Port:             "9998",
		ShopifyAPIKey:    apiKey,
		ShopifyAPISecret: apiSecret,
		DebugAuth:        true,
	}

	var buf bytes.Buffer
	saved := logger.Log
	logger.Log = zerolog.New(&buf)
	t.Cleanup(func() { logger.Log = saved })

	srv := newTestServer(t, cfg)

	token := makeValidShopifySessionToken(t, apiKey, apiSecret, shopDomain)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/admin/ping", nil)
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

	result := getJSON(t, resp)
	shop, _ := result["shop"].(string)
	if shop != shopDomain {
		t.Fatalf("shop: got %q, want %q", shop, shopDomain)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "shopify_session_token") {
		t.Fatalf("expected claims log, got %q", logOutput)
	}
}

func TestAdminPingShopMismatchClearsContext(t *testing.T) {
	const apiKey = "test-key"
	const apiSecret = "test-secret"
	const jwtShop = "test-shop.myshopify.com"
	const headerShop = "other-shop.myshopify.com"

	cfg := config.Config{
		Port:             "9998",
		ShopifyAPIKey:    apiKey,
		ShopifyAPISecret: apiSecret,
	}
	srv := newTestServer(t, cfg)

	token := makeValidShopifySessionToken(t, apiKey, apiSecret, jwtShop)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/admin/ping", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Shop-Domain", headerShop)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /admin/ping: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status: got %d, want %d; body=%s", resp.StatusCode, http.StatusOK, string(body))
	}

	result := getJSON(t, resp)
	shop, _ := result["shop"].(string)
	if shop != "" {
		t.Fatalf("shop should be empty on mismatch, got %q", shop)
	}
}

func TestAdminPingShopMatchKeepsContext(t *testing.T) {
	const apiKey = "test-key"
	const apiSecret = "test-secret"
	const shopDomain = "test-shop.myshopify.com"

	cfg := config.Config{
		Port:             "9998",
		ShopifyAPIKey:    apiKey,
		ShopifyAPISecret: apiSecret,
	}
	srv := newTestServer(t, cfg)

	token := makeValidShopifySessionToken(t, apiKey, apiSecret, shopDomain)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/admin/ping", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Shop-Domain", shopDomain)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /admin/ping: %v", err)
	}
	defer resp.Body.Close()

	result := getJSON(t, resp)
	shop, _ := result["shop"].(string)
	if shop != shopDomain {
		t.Fatalf("shop: got %q, want %q", shop, shopDomain)
	}
}

func makeSignedAppProxyQuery(secret, shopDomain string) string {
	values := url.Values{}
	values.Set("shop", shopDomain)
	values.Set("logged_in_customer_id", "89466365215417")
	values.Set("path_prefix", "/apps/my-app-test")
	values.Set("timestamp", "1772420951")

	signature := mw.ComputeSHA256HMACHex(mw.CanonicalizeProxyParams(values), secret)
	values.Set("signature", signature)
	return values.Encode()
}

func TestAppProxyRouteRequiresValidSignature(t *testing.T) {
	const apiSecret = "test-secret"
	const shopDomain = "devloop-4.myshopify.com"

	srv := newTestServer(t, defaultTestConfig())

	resp, err := http.Get(srv.URL + "/app/ping")
	if err != nil {
		t.Fatalf("GET /app/ping: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}

	query := makeSignedAppProxyQuery(apiSecret, shopDomain)
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

func TestAppProxyPingReturnsShop(t *testing.T) {
	const apiSecret = "test-secret"
	const shopDomain = "devloop-4.myshopify.com"

	srv := newTestServer(t, defaultTestConfig())

	query := makeSignedAppProxyQuery(apiSecret, shopDomain)
	resp, err := http.Get(srv.URL + "/app/ping?" + query)
	if err != nil {
		t.Fatalf("GET /app/ping: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status: got %d, want %d; body=%s", resp.StatusCode, http.StatusOK, string(body))
	}

	result := getJSON(t, resp)
	shop, _ := result["shop"].(string)
	if shop != shopDomain {
		t.Fatalf("shop: got %q, want %q", shop, shopDomain)
	}
}

func TestAppProxyPingShopMismatchClearsContext(t *testing.T) {
	const apiSecret = "test-secret"
	const proxyShop = "devloop-4.myshopify.com"
	const headerShop = "other-shop.myshopify.com"

	srv := newTestServer(t, defaultTestConfig())

	query := makeSignedAppProxyQuery(apiSecret, proxyShop)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/app/ping?"+query, nil)
	req.Header.Set("X-Shop-Domain", headerShop)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /app/ping: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status: got %d, want %d; body=%s", resp.StatusCode, http.StatusOK, string(body))
	}

	result := getJSON(t, resp)
	shop, _ := result["shop"].(string)
	if shop != "" {
		t.Fatalf("shop should be empty on mismatch, got %q", shop)
	}
}
