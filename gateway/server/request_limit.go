package server

import (
	"Hamburger/internal/config/core_config"
	"net/http"

	"github.com/rs/zerolog"
)

// WrapHandlerWithMaxQuerySize rejects requests whose complete request target
// (the raw path plus query string) exceeds the configured byte limit. The
// limit is resolved once when the listener is built so a configuration reload
// takes effect on the following gateway restart.
func WrapHandlerWithMaxQuerySize(h http.Handler, configured int64, logger *zerolog.Logger) (http.Handler, error) {
	maxQuerySize, err := core_config.EffectiveMaxQuerySize(configured)
	if err != nil {
		return nil, err
	}

	if logger != nil {
		logger.Debug().Int64("max_query_size", maxQuerySize).Msg("gateway request target size limit configured")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestTargetSize := int64(len(r.RequestURI))
		if requestTargetSize > maxQuerySize {
			if r.Body != nil {
				_ = r.Body.Close()
			}
			if logger != nil {
				logger.Debug().
					Str("method", r.Method).
					Str("host", r.Host).
					Str("remote_addr", r.RemoteAddr).
					Int64("request_target_size", requestTargetSize).
					Int64("max_query_size", maxQuerySize).
					Msg("request target too long")
			}
			http.Error(w, "Request URI too long", http.StatusRequestURITooLong)
			return
		}

		h.ServeHTTP(w, r)
	}), nil
}
