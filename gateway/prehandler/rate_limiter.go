package prehandler

import (
	flow "Hamburger/gateway/flow_control"
	"Hamburger/internal/config"
	"Hamburger/internal/logger"
	"Hamburger/internal/serror"
	"net/http"
)

type RateLimiter struct {
	enabled bool
	fc      *flow.FlowController
}

func NewRateLimiter() *RateLimiter {
	cf := config.Get()
	flowController := flow.NewFlowController()
	go flowController.CleanupExpiredRecords()
	return &RateLimiter{
		enabled: cf.Features.FlowControl.Enabled,
		fc:      flowController,
	}
}

func (r RateLimiter) Handle(req *http.Request) error {
	if !r.Enabled() {
		return nil
	}
	if r.fc != nil {
		result := r.fc.CheckRequest(req)
		if !result.Allowed {
			// 记录被限流的请求
			flowRecorder := flow.GetFlowRecorder()
			if flowRecorder != nil {
				flowRecorder.RecordBlocked(req, result)
			}

			logger.GetLogger().Debug().
				Str("Host", req.Host).
				Str("Method", req.Method).
				Str("Remote Addr", req.RemoteAddr).
				Str("Reason", result.Reason).
				Msg("client has been rate limited")
			req.Header.Set(serror.SandwichInternalFlag, serror.SandwichReqLimit)
		} else {
			// 记录通过的请求（如果启用）
			flowRecorder := flow.GetFlowRecorder()
			if flowRecorder != nil {
				flowRecorder.RecordAllowed(req)
			}
		}
	}

	return nil
}

func (r RateLimiter) Name() string {
	return "RateLimiter"
}

func (r RateLimiter) Enabled() bool {
	return r.enabled
}
