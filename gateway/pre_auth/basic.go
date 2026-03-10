package pre_auth

import (
	"Hamburger/internal/config"
	"net/http"
)

type BasicAuthenticator struct {
	enabled bool
	users   map[string]string
}

func NewBasicAuthenticator(cfg config.BasicAuthConfig) *BasicAuthenticator {
	users := make(map[string]string, len(cfg.Users))
	for _, user := range cfg.Users {
		if user.Username == "" {
			continue
		}
		users[user.Username] = user.Password
	}
	return &BasicAuthenticator{
		enabled: cfg.Enabled,
		users:   users,
	}
}

func (a *BasicAuthenticator) Authenticate(req *http.Request) (Result, error) {
	if !a.enabled {
		return Result{}, nil
	}
	username, password, ok := req.BasicAuth()
	if !ok {
		return Result{Valid: false}, nil
	}
	if expectedPassword, exists := a.users[username]; !exists || expectedPassword != password {
		return Result{Valid: false}, nil
	}
	return Result{
		Valid:  true,
		Method: a.Name(),
		Token:  username,
	}, nil
}

func (a *BasicAuthenticator) Name() string {
	return "basic"
}

func (a *BasicAuthenticator) Enabled() bool {
	return a.enabled
}
