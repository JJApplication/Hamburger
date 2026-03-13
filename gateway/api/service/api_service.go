package service

import (
	"Hamburger/gateway/api/model"
	"Hamburger/internal/config"
	"errors"
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
