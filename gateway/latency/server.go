package latency

import (
	"Hamburger/internal/config"
	"Hamburger/internal/config/loader"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// latency server

type LatencyServer struct {
	Enabled bool
	Addr    string
	Port    int

	logger *zerolog.Logger
	server *http.Server
}

func NewLatencyServer(c config.LatencyConfig, l *zerolog.Logger) *LatencyServer {
	if !c.Enabled {
		return new(LatencyServer)
	}
	return &LatencyServer{
		Enabled: c.Enabled,
		Addr:    c.Host,
		Port:    c.Port,

		logger: l,
		server: newServer(c.Host, c.Port),
	}
}

func (s *LatencyServer) Start() error {
	if !s.Enabled {
		return nil
	}
	s.logger.Info().Str("address", s.Addr).Int("port", s.Port).Msg("start latency server")

	go func() {
		err := s.server.ListenAndServe()
		if err != nil {
			s.logger.Error().Err(err).Msg("latency server listen err")
		}
	}()

	return nil
}

func (s *LatencyServer) Stop() error {
	if !s.Enabled {
		return nil
	}
	return s.server.Shutdown(context.Background())
}

func newServer(host string, port int) *http.Server {
	svr := &http.Server{
		Addr: fmt.Sprintf("%s:%d", host, port),
	}
	mux := http.NewServeMux()

	registerMux(mux)
	svr.Handler = mux
	return svr
}

func registerMux(mux *http.ServeMux) {
	mux.HandleFunc("/api/latency", func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIPFromRequest(r)
		testDomainAddress := r.URL.Query().Get("domain")

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if IsInBlackList(clientIP) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if recordLatencyRequest(clientIP, 60*time.Second, 10) {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		normalizedURL, host, err := normalizeDomain(testDomainAddress)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		cfg := loader.Get()
		if cfg != nil {
			if isDomainInList(host, normalizedURL, cfg.Latency.DomainBlackList) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			if len(cfg.Latency.DomainWhiteList) > 0 && !isDomainInList(host, normalizedURL, cfg.Latency.DomainWhiteList) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
		}

		start := time.Now()
		response, err := doLatencyRequest(normalizedURL)
		if err != nil {
			resp := LatencyResponse{
				Domain:       host,
				Normalized:   normalizedURL,
				StatusCode:   0,
				DurationMs:   time.Since(start).Milliseconds(),
				BodySize:     0,
				ErrorMessage: err.Error(),
			}
			writeLatencyResponse(w, http.StatusInternalServerError, resp)
			return
		}
		response.DurationMs = time.Since(start).Milliseconds()
		writeLatencyResponse(w, http.StatusOK, response)
	})
}
