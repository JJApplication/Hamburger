package pre_auth

import "net/http"

const (
	GatewayAuthTokenHeader  = "X-Gateway-Auth-Token"
	GatewayAuthMethodHeader = "X-Gateway-Auth-Method"
)

type Result struct {
	Valid  bool
	Method string
	Token  string
}

type Authenticator interface {
	Authenticate(*http.Request) (Result, error)
	Name() string
	Enabled() bool
}
