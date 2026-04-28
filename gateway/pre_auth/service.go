package pre_auth

import (
	"Hamburger/internal/config/svr_config"
	"errors"
	"net/http"
	"strings"
)

var ErrUnauthorized = errors.New("pre auth unauthorized")

type Service struct {
	enabled           bool
	passThroughHeader string
	passThroughValue  string
	authenticators    []Authenticator
}

func NewService(cfg svr_config.PreAuthConfig) *Service {
	authenticators := []Authenticator{
		NewOAuth2Authenticator(cfg.OAuth2),
		NewBasicAuthenticator(cfg.Basic),
		NewJWTAuthenticator(cfg.JWT),
	}
	return &Service{
		enabled:           cfg.Enabled,
		passThroughHeader: strings.TrimSpace(cfg.PassThroughHeader),
		passThroughValue:  strings.TrimSpace(cfg.PassThroughValue),
		authenticators:    authenticators,
	}
}

func (s *Service) Enabled() bool {
	if !s.enabled {
		return false
	}
	for _, auth := range s.authenticators {
		if auth.Enabled() {
			return true
		}
	}
	return false
}

func (s *Service) Authenticate(req *http.Request) (Result, error) {
	if !s.Enabled() {
		return Result{}, nil
	}
	for _, auth := range s.authenticators {
		if !auth.Enabled() {
			continue
		}
		result, err := auth.Authenticate(req)
		if err != nil {
			continue
		}
		if result.Valid {
			s.applyHeaders(req, result)
			return result, nil
		}
	}
	return Result{Valid: false}, ErrUnauthorized
}

func (s *Service) applyHeaders(req *http.Request, result Result) {
	if req == nil {
		return
	}
	req.Header.Set(GatewayAuthTokenHeader, result.Token)
	req.Header.Set(GatewayAuthMethodHeader, result.Method)
	if s.passThroughHeader == "" {
		return
	}
	value := s.passThroughValue
	if value == "" {
		value = "verified"
	}
	req.Header.Set(s.passThroughHeader, value)
}
