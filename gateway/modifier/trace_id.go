package modifier

import (
	"Hamburger/internal/config"
	"Hamburger/internal/utils"
	"net/http"
)

type TraceModifier struct {
	enable bool
	header string
}

func NewTraceModifier() *TraceModifier {
	cfg := config.Get()

	mod := new(TraceModifier)
	mod.enable = cfg.Middleware.Trace.Enabled
	mod.header = cfg.Middleware.Trace.TraceId
	return mod
}

func (t TraceModifier) Use(response *http.Response) {
	if !t.enable {
		return
	}
	utils.AddTrace(response, t.header)
}

func (t TraceModifier) ModifyResponse(response *http.Response) error {
	return nil
}

func (t TraceModifier) IsEnabled() bool {
	return t.enable
}

func (t TraceModifier) UpdateConfig() {
	return
}

func (t TraceModifier) GetName() string {
	return "trace-id"
}
