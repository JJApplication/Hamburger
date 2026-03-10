package pre_auth

import (
	"Hamburger/internal/config"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBasicAuthenticatorSuccess(t *testing.T) {
	auth := NewBasicAuthenticator(config.BasicAuthConfig{
		Enabled: true,
		Users: []config.BasicUserConfig{
			{Username: "admin", Password: "123456"},
		},
	})
	req := httptest.NewRequest(http.MethodGet, "http://localhost/test", nil)
	req.SetBasicAuth("admin", "123456")
	result, err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid || result.Method != "basic" || result.Token != "admin" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestJWTAuthenticatorSuccess(t *testing.T) {
	secret := "secret-key"
	token := buildJWT(t, secret, map[string]interface{}{
		"iss": "gateway",
		"aud": "backend",
		"exp": time.Now().Add(5 * time.Minute).Unix(),
	})
	auth := NewJWTAuthenticator(config.JWTAuthConfig{
		Enabled:        true,
		TokenHeader:    "Authorization",
		Secret:         secret,
		Issuer:         "gateway",
		Audience:       "backend",
		AllowedMethods: []string{"HS256"},
	})
	req := httptest.NewRequest(http.MethodGet, "http://localhost/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	result, err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid || result.Method != "jwt" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestOAuth2AuthenticatorStaticToken(t *testing.T) {
	auth := NewOAuth2Authenticator(config.OAuth2AuthConfig{
		Enabled:      true,
		TokenHeader:  "Authorization",
		AccessTokens: []string{"valid-token"},
	})
	req := httptest.NewRequest(http.MethodGet, "http://localhost/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	result, err := auth.Authenticate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid || result.Method != "oauth2" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestServiceAuthenticateSetsHeaders(t *testing.T) {
	service := NewService(config.PreAuthConfig{
		Enabled:           true,
		PassThroughHeader: "X-Backend-Bypass-Auth",
		PassThroughValue:  "allow",
		Basic: config.BasicAuthConfig{
			Enabled: true,
			Users: []config.BasicUserConfig{
				{Username: "svc", Password: "pwd"},
			},
		},
	})
	req := httptest.NewRequest(http.MethodGet, "http://localhost/test", nil)
	req.SetBasicAuth("svc", "pwd")
	result, err := service.Authenticate(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid result")
	}
	if req.Header.Get(GatewayAuthMethodHeader) != "basic" {
		t.Fatalf("missing auth method header")
	}
	if req.Header.Get(GatewayAuthTokenHeader) != "svc" {
		t.Fatalf("missing auth token header")
	}
	if req.Header.Get("X-Backend-Bypass-Auth") != "allow" {
		t.Fatalf("missing pass through header")
	}
}

func buildJWT(t *testing.T, secret string, claims map[string]interface{}) string {
	t.Helper()
	headerBytes, err := json.Marshal(map[string]interface{}{
		"alg": "HS256",
		"typ": "JWT",
	})
	if err != nil {
		t.Fatalf("marshal header failed: %v", err)
	}
	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}
	h := base64.RawURLEncoding.EncodeToString(headerBytes)
	p := base64.RawURLEncoding.EncodeToString(payloadBytes)
	input := h + "." + p
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(input))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return input + "." + sig
}
