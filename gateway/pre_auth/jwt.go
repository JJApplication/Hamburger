package pre_auth

import (
	"Hamburger/internal/config/svr_config"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"net/http"
	"strings"
	"time"
)

type JWTAuthenticator struct {
	enabled        bool
	tokenHeader    string
	secret         []byte
	issuer         string
	audience       string
	allowedMethods map[string]func() hash.Hash
}

func NewJWTAuthenticator(cfg svr_config.JWTAuthConfig) *JWTAuthenticator {
	methods := make(map[string]func() hash.Hash)
	for _, method := range cfg.AllowedMethods {
		method = strings.TrimSpace(method)
		if method == "" {
			continue
		}
		if h := jwtHashMethod(method); h != nil {
			methods[method] = h
		}
	}
	if len(methods) == 0 {
		methods["HS256"] = sha256.New
	}
	return &JWTAuthenticator{
		enabled:        cfg.Enabled,
		tokenHeader:    cfg.TokenHeader,
		secret:         []byte(cfg.Secret),
		issuer:         strings.TrimSpace(cfg.Issuer),
		audience:       strings.TrimSpace(cfg.Audience),
		allowedMethods: methods,
	}
}

func (a *JWTAuthenticator) Authenticate(req *http.Request) (Result, error) {
	if !a.enabled {
		return Result{}, nil
	}
	if len(a.secret) == 0 {
		return Result{Valid: false}, nil
	}
	token, ok := extractBearerToken(req, a.tokenHeader)
	if !ok {
		return Result{Valid: false}, nil
	}
	claims, err := a.validateToken(token)
	if err != nil {
		return Result{Valid: false}, nil
	}
	if err = a.validateClaims(claims); err != nil {
		return Result{Valid: false}, nil
	}
	return Result{
		Valid:  true,
		Method: a.Name(),
		Token:  token,
	}, nil
}

func (a *JWTAuthenticator) validateToken(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid jwt format")
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}
	var header map[string]interface{}
	if err = json.Unmarshal(headerBytes, &header); err != nil {
		return nil, err
	}
	alg, _ := header["alg"].(string)
	hashFunc, ok := a.allowedMethods[alg]
	if !ok {
		return nil, fmt.Errorf("unsupported alg")
	}
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(hashFunc, a.secret)
	_, _ = mac.Write([]byte(signingInput))
	expected := mac.Sum(nil)
	if !hmac.Equal(signature, expected) {
		return nil, fmt.Errorf("signature mismatch")
	}
	var claims map[string]interface{}
	if err = json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func (a *JWTAuthenticator) validateClaims(claims map[string]interface{}) error {
	now := time.Now().Unix()
	if exp, ok := int64Claim(claims["exp"]); ok && now >= exp {
		return fmt.Errorf("token expired")
	}
	if nbf, ok := int64Claim(claims["nbf"]); ok && now < nbf {
		return fmt.Errorf("token not active")
	}
	if a.issuer != "" {
		iss, _ := claims["iss"].(string)
		if iss != a.issuer {
			return fmt.Errorf("issuer mismatch")
		}
	}
	if a.audience != "" {
		if !containsAudience(claims["aud"], a.audience) {
			return fmt.Errorf("audience mismatch")
		}
	}
	return nil
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

func (a *JWTAuthenticator) Name() string {
	return "jwt"
}

func (a *JWTAuthenticator) Enabled() bool {
	return a.enabled
}
