package service

import (
	"Hamburger/gateway/api/model"
	"Hamburger/internal/config"
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
	jwtCfg     config.JWTConfig
	bboltCfg   config.APIBBoltConfig
	db         *bbolt.DB
	userBucket []byte
	initErr    error
	stopFn     map[string]func() error
	restartFn  map[string]func() error
}

func NewAPIService(apiCfg config.ApiServerConfig) *APIService {
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
	key := strings.TrimSpace(strings.ToLower(name))
	if key == "" {
		return fmt.Errorf("server is empty")
	}
	if s.stopFn == nil {
		return fmt.Errorf("server control unavailable")
	}
	fn, ok := s.stopFn[key]
	if !ok || fn == nil {
		return fmt.Errorf("unsupported server: %s", key)
	}
	return fn()
}

func (s *APIService) RestartServer(name string) error {
	key := strings.TrimSpace(strings.ToLower(name))
	if key == "" {
		return fmt.Errorf("server is empty")
	}
	if s.restartFn == nil {
		return fmt.Errorf("server control unavailable")
	}
	fn, ok := s.restartFn[key]
	if !ok || fn == nil {
		return fmt.Errorf("unsupported server: %s", key)
	}
	return fn()
}
