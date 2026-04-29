package pre_auth

import (
	"Hamburger/internal/config/svr_config"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type OAuth2Authenticator struct {
	enabled          bool
	tokenHeader      string
	introspectionURL string
	clientID         string
	clientSecret     string
	requiredScopes   map[string]struct{}
	accessTokens     map[string]struct{}
	client           *http.Client
}

func NewOAuth2Authenticator(cfg svr_config.OAuth2AuthConfig) *OAuth2Authenticator {
	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = 3
	}
	requiredScopes := make(map[string]struct{}, len(cfg.RequiredScopes))
	for _, scope := range cfg.RequiredScopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			continue
		}
		requiredScopes[scope] = struct{}{}
	}
	accessTokens := make(map[string]struct{}, len(cfg.AccessTokens))
	for _, token := range cfg.AccessTokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		accessTokens[token] = struct{}{}
	}
	return &OAuth2Authenticator{
		enabled:          cfg.Enabled,
		tokenHeader:      cfg.TokenHeader,
		introspectionURL: strings.TrimSpace(cfg.IntrospectionURL),
		clientID:         cfg.ClientID,
		clientSecret:     cfg.ClientSecret,
		requiredScopes:   requiredScopes,
		accessTokens:     accessTokens,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

func (a *OAuth2Authenticator) Authenticate(req *http.Request) (Result, error) {
	if !a.enabled {
		return Result{}, nil
	}
	token, ok := extractBearerToken(req, a.tokenHeader)
	if !ok {
		return Result{Valid: false}, nil
	}
	if len(a.accessTokens) > 0 {
		if _, exists := a.accessTokens[token]; exists {
			return Result{
				Valid:  true,
				Method: a.Name(),
				Token:  token,
			}, nil
		}
	}
	if a.introspectionURL == "" {
		return Result{Valid: false}, nil
	}
	valid, err := a.validateByIntrospection(token)
	if err != nil {
		return Result{Valid: false}, err
	}
	if !valid {
		return Result{Valid: false}, nil
	}
	return Result{
		Valid:  true,
		Method: a.Name(),
		Token:  token,
	}, nil
}

func (a *OAuth2Authenticator) validateByIntrospection(token string) (bool, error) {
	values := url.Values{}
	values.Set("token", token)
	req, err := http.NewRequest(http.MethodPost, a.introspectionURL, bytes.NewBufferString(values.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if a.clientID != "" {
		req.SetBasicAuth(a.clientID, a.clientSecret)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return false, fmt.Errorf("oauth2 introspection status %d", resp.StatusCode)
	}
	var payload struct {
		Active bool   `json:"active"`
		Scope  string `json:"scope"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return false, err
	}
	if !payload.Active {
		return false, nil
	}
	if len(a.requiredScopes) == 0 {
		return true, nil
	}
	scopes := strings.Fields(payload.Scope)
	scopeSet := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		scopeSet[scope] = struct{}{}
	}
	for scope := range a.requiredScopes {
		if _, ok := scopeSet[scope]; !ok {
			return false, nil
		}
	}
	return true, nil
}

func (a *OAuth2Authenticator) Name() string {
	return "oauth2"
}

func (a *OAuth2Authenticator) Enabled() bool {
	return a.enabled
}
