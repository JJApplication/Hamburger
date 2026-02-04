package prehandler

import "net/http"

type PreHandler interface {
	Handle(*http.Request) error
	Name() string
	Enabled() bool
}
