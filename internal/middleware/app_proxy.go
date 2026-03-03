package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// ShopifyAppProxySignatureMiddleware validates the Shopify app proxy HMAC
// signature. On success it reconciles the shop domain from query params
// with the header-based shop in the request context.
func ShopifyAppProxySignatureMiddleware(apiSecret string, debugAuth bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestPath := r.URL.Path
			queryString := strings.TrimSpace(r.Header.Get("hmac"))
			payloadSource := "header:hmac"
			if queryString == "" {
				queryString = r.URL.RawQuery
				payloadSource = "url:query"
			}
			if queryString == "" {
				if debugAuth {
					log.Printf("app_proxy_hmac: missing payload path=%s", requestPath)
				}
				http.Error(w, "missing hmac payload", http.StatusUnauthorized)
				return
			}

			if idx := strings.Index(queryString, "?"); idx >= 0 && idx+1 < len(queryString) {
				queryString = queryString[idx+1:]
			}

			values, err := url.ParseQuery(queryString)
			if err != nil {
				if debugAuth {
					log.Printf("app_proxy_hmac: parse payload failed path=%s source=%s err=%v", requestPath, payloadSource, err)
				}
				http.Error(w, "invalid hmac payload", http.StatusUnauthorized)
				return
			}

			signature := strings.TrimSpace(values.Get("signature"))
			if signature == "" {
				signature = strings.TrimSpace(values.Get("hmac"))
			}
			if signature == "" {
				if debugAuth {
					log.Printf("app_proxy_hmac: missing signature path=%s source=%s", requestPath, payloadSource)
				}
				http.Error(w, "missing signature", http.StatusUnauthorized)
				return
			}

			values.Del("signature")
			values.Del("hmac")

			message := CanonicalizeProxyParams(values)
			expected := ComputeSHA256HMACHex(message, apiSecret)
			if !hmac.Equal([]byte(strings.ToLower(signature)), []byte(expected)) {
				if debugAuth {
					log.Printf(
						"app_proxy_hmac: signature mismatch path=%s source=%s keys=%d got_prefix=%s expected_prefix=%s",
						requestPath,
						payloadSource,
						len(values),
						shortHex(signature),
						shortHex(expected),
					)
				}
				http.Error(w, "invalid signature", http.StatusUnauthorized)
				return
			}

			if debugAuth {
				log.Printf("app_proxy_hmac: verified path=%s source=%s keys=%d", requestPath, payloadSource, len(values))
			}

			ctx := r.Context()
			if shopParam := strings.TrimSpace(values.Get("shop")); shopParam != "" {
				if shopDomain, err := extractShopDomain(shopParam); err == nil {
					ctx = reconcileShopContext(ctx, shopDomain)
				}
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func shortHex(s string) string {
	if len(s) <= 8 {
		return s
	}
	return s[:8]
}

// CanonicalizeProxyParams builds the canonical string for HMAC verification.
func CanonicalizeProxyParams(values url.Values) string {
	pairs := make([]string, 0, len(values))
	for key, vals := range values {
		pairs = append(pairs, key+"="+strings.Join(vals, ","))
	}
	sort.Strings(pairs)
	return strings.Join(pairs, "")
}

// ComputeSHA256HMACHex returns the hex-encoded HMAC-SHA256 of message with the given secret.
func ComputeSHA256HMACHex(message, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}
