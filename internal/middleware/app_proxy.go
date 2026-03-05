package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"shopify-app-authentication/internal/logger"
)

// ShopifyAppProxySignatureMiddleware 校验 Shopify App Proxy 请求的 HMAC 签名。
// 签名来源优先从请求头 hmac 读取，其次从 URL query 读取。
// 校验通过后将 query 中的 shop 参数与上下文中的店铺域名进行一致性对比。
func ShopifyAppProxySignatureMiddleware(apiSecret string, debugAuth bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestPath := r.URL.Path

			// 1. 获取签名载荷（优先 header，其次 URL query）
			queryString := strings.TrimSpace(r.Header.Get("hmac"))
			payloadSource := "header:hmac"
			if queryString == "" {
				queryString = r.URL.RawQuery
				payloadSource = "url:query"
			}
			if queryString == "" {
				if debugAuth {
					logger.Log.Debug().Str("path", requestPath).Msg("app_proxy_hmac: missing payload")
				}
				http.Error(w, "missing hmac payload", http.StatusUnauthorized)
				return
			}

			// 如果载荷包含 "?" 前缀（来自完整 URL），只取后半段
			if idx := strings.Index(queryString, "?"); idx >= 0 && idx+1 < len(queryString) {
				queryString = queryString[idx+1:]
			}

			// 2. 解析参数
			values, err := url.ParseQuery(queryString)
			if err != nil {
				if debugAuth {
					logger.Log.Warn().Str("path", requestPath).Str("source", payloadSource).Err(err).
						Msg("app_proxy_hmac: parse payload failed")
				}
				http.Error(w, "invalid hmac payload", http.StatusUnauthorized)
				return
			}

			// 3. 提取 signature（兼容 hmac 字段名）
			signature := strings.TrimSpace(values.Get("signature"))
			if signature == "" {
				signature = strings.TrimSpace(values.Get("hmac"))
			}
			if signature == "" {
				if debugAuth {
					logger.Log.Debug().Str("path", requestPath).Str("source", payloadSource).
						Msg("app_proxy_hmac: missing signature")
				}
				http.Error(w, "missing signature", http.StatusUnauthorized)
				return
			}

			// 4. 用剩余参数计算期望签名并比对
			values.Del("signature")
			values.Del("hmac")

			message := CanonicalizeProxyParams(values)
			expected := ComputeSHA256HMACHex(message, apiSecret)
			if !hmac.Equal([]byte(strings.ToLower(signature)), []byte(expected)) {
				if debugAuth {
					logger.Log.Warn().Str("path", requestPath).Str("source", payloadSource).
						Int("keys", len(values)).
						Str("got_prefix", shortHex(signature)).Str("expected_prefix", shortHex(expected)).
						Msg("app_proxy_hmac: signature mismatch")
				}
				http.Error(w, "invalid signature", http.StatusUnauthorized)
				return
			}

			if debugAuth {
				logger.Log.Info().Str("path", requestPath).Str("source", payloadSource).
					Int("keys", len(values)).Msg("app_proxy_hmac: verified")
			}

			// 5. 将 query 中的 shop 参数与请求头中的店铺域名进行一致性校验
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

// shortHex 返回十六进制字符串的前 8 位，用于日志脱敏。
func shortHex(s string) string {
	if len(s) <= 8 {
		return s
	}
	return s[:8]
}

// CanonicalizeProxyParams 将参数按 key 字母序排列后拼接成签名用的规范字符串。
// 格式: key1=val1key2=val2（无分隔符），符合 Shopify App Proxy 签名规范。
func CanonicalizeProxyParams(values url.Values) string {
	pairs := make([]string, 0, len(values))
	for key, vals := range values {
		pairs = append(pairs, key+"="+strings.Join(vals, ","))
	}
	sort.Strings(pairs)
	return strings.Join(pairs, "")
}

// ComputeSHA256HMACHex 使用 HMAC-SHA256 对 message 签名，返回十六进制编码结果。
func ComputeSHA256HMACHex(message, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}
