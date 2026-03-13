package middleware

import (
	"Hamburger/internal/config"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"hash"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func JWT(cfg config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}
		token, ok := extractJWTToken(c.Request, cfg.TokenHeader)
		if !ok || !validateJWTToken(token, cfg) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}
		c.Next()
	}
}

func extractJWTToken(req *http.Request, headerName string) (string, bool) {
	key := strings.TrimSpace(headerName)
	if key == "" {
		key = "Authorization"
	}
	value := strings.TrimSpace(req.Header.Get(key))
	if value == "" {
		return "", false
	}
	if strings.HasPrefix(strings.ToLower(value), "bearer ") {
		token := strings.TrimSpace(value[7:])
		return token, token != ""
	}
	return value, true
}

func validateJWTToken(token string, cfg config.JWTConfig) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	var header map[string]interface{}
	if err = json.Unmarshal(headerBytes, &header); err != nil {
		return false
	}
	alg, _ := header["alg"].(string)
	hashFunc := jwtHashMethod(alg)
	if hashFunc == nil {
		return false
	}
	if !isAllowedMethod(alg, cfg.AllowedMethods) {
		return false
	}
	secret := []byte(cfg.Secret)
	if len(secret) == 0 {
		return false
	}
	mac := hmac.New(hashFunc, secret)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return false
	}
	var claims map[string]interface{}
	if err = json.Unmarshal(payloadBytes, &claims); err != nil {
		return false
	}
	return validateJWTClaims(claims, strings.TrimSpace(cfg.Issuer), strings.TrimSpace(cfg.Audience))
}

func validateJWTClaims(claims map[string]interface{}, issuer string, audience string) bool {
	now := time.Now().Unix()
	if exp, ok := int64Claim(claims["exp"]); ok && now >= exp {
		return false
	}
	if nbf, ok := int64Claim(claims["nbf"]); ok && now < nbf {
		return false
	}
	if issuer != "" {
		iss, _ := claims["iss"].(string)
		if iss != issuer {
			return false
		}
	}
	if audience != "" && !containsAudience(claims["aud"], audience) {
		return false
	}
	return true
}

func int64Claim(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case float64:
		return int64(v), true
	case int64:
		return v, true
	case int:
		return int64(v), true
	default:
		return 0, false
	}
}

func containsAudience(value interface{}, audience string) bool {
	switch aud := value.(type) {
	case string:
		return aud == audience
	case []interface{}:
		for _, item := range aud {
			if str, ok := item.(string); ok && str == audience {
				return true
			}
		}
	}
	return false
}

func jwtHashMethod(method string) func() hash.Hash {
	switch method {
	case "HS256":
		return sha256.New
	case "HS384":
		return sha512.New384
	case "HS512":
		return sha512.New
	default:
		return nil
	}
}

func isAllowedMethod(method string, allowed []string) bool {
	if len(allowed) == 0 {
		return method == "HS256"
	}
	for _, item := range allowed {
		if strings.TrimSpace(item) == method {
			return true
		}
	}
	return false
}
