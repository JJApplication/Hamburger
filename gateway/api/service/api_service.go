package service

import (
	"Hamburger/gateway/api/model"
	"Hamburger/internal/config/svr_config"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.etcd.io/bbolt"
)

var (
	errUserNotFound     = errors.New("user not found")
	errInvalidPassword  = errors.New("invalid password")
	errInvalidToken     = errors.New("invalid token")
	errStoreUnavailable = errors.New("user store unavailable")
)

type APIService struct {
	jwtCfg     svr_config.JWTConfig
	bboltCfg   svr_config.APIBBoltConfig
	db         *bbolt.DB
	userBucket []byte
	initErr    error
	stopFn     map[string]func() error
	restartFn  map[string]func() error
}

func NewAPIService(apiCfg svr_config.ApiServerConfig) *APIService {
	s := &APIService{
		jwtCfg:   apiCfg.JWT,
		bboltCfg: apiCfg.BBolt,
	}
	if !s.bboltCfg.Enabled {
		return s
	}
	if s.bboltCfg.File == "" {
		s.bboltCfg.File = model.DefaultBBLOTName
	}
	if s.bboltCfg.TimeoutSeconds <= 0 {
		s.bboltCfg.TimeoutSeconds = 1
	}
	if strings.TrimSpace(s.bboltCfg.UserBucket) == "" {
		s.bboltCfg.UserBucket = model.BucketUser
	}
	db, err := bbolt.Open(s.bboltCfg.File, 0600, &bbolt.Options{
		Timeout: time.Duration(s.bboltCfg.TimeoutSeconds) * time.Second,
	})
	if err != nil {
		s.initErr = err
		return s
	}
	s.db = db
	s.userBucket = []byte(s.bboltCfg.UserBucket)
	s.initErr = s.initDefaultUser()
	return s
}

func (s *APIService) CloseDB() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *APIService) SetServerControl(stopFn map[string]func() error, restartFn map[string]func() error) {
	s.stopFn = stopFn
	s.restartFn = restartFn
}

func (s *APIService) StopServer(name string) error {
	fn, err := s.serverControl(s.stopFn, name)
	if err != nil {
		return err
	}
	return fn()
}

func (s *APIService) RestartServer(name string) error {
	fn, err := s.serverControl(s.restartFn, name)
	if err != nil {
		return err
	}
	return fn()
}

// StopServerAsync starts a server stop after the current request has returned.
// It is used by Connect's gateway control method because waiting for a gateway
// shutdown from a request served by that gateway would wait on itself.
func (s *APIService) StopServerAsync(name string) error {
	fn, err := s.serverControl(s.stopFn, name)
	if err != nil {
		return err
	}
	go func() { _ = fn() }()
	return nil
}

// RestartServerAsync is the non blocking counterpart of RestartServer.
func (s *APIService) RestartServerAsync(name string) error {
	fn, err := s.serverControl(s.restartFn, name)
	if err != nil {
		return err
	}
	go func() { _ = fn() }()
	return nil
}

func (s *APIService) serverControl(controls map[string]func() error, name string) (func() error, error) {
	key := strings.TrimSpace(strings.ToLower(name))
	if key == "" {
		return nil, fmt.Errorf("server is empty")
	}
	if controls == nil {
		return nil, fmt.Errorf("server control unavailable")
	}
	fn, ok := controls[key]
	if !ok || fn == nil {
		return nil, fmt.Errorf("unsupported server: %s", key)
	}
	return fn, nil
}
