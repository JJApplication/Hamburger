package service

import (
	"Hamburger/gateway/api/model"
	"Hamburger/internal/config/loader"
	"Hamburger/internal/constant"
	"Hamburger/internal/json"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
	"hash"
	"net/http"
	"strings"
	"time"
)

func (s *APIService) Login(username, password string) (string, model.User, error) {
	record, err := s.loadUser(username)
	if err != nil {
		return "", model.User{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(record.PasswordHash), []byte(password)) != nil {
		return "", model.User{}, errInvalidPassword
	}
	token, err := s.createToken(record.Username)
	if err != nil {
		return "", model.User{}, err
	}
	return token, toUser(record), nil
}

func (s *APIService) Logout() error {
	return nil
}

func (s *APIService) GetUserByToken(token string) (model.User, error) {
	username, err := s.usernameFromToken(token)
	if err != nil {
		return model.User{}, err
	}
	record, err := s.loadUser(username)
	if err != nil {
		return model.User{}, err
	}
	return toUser(record), nil
}

func (s *APIService) UpdateUserByToken(token string, username, nickname, avatar, password string) (model.User, error) {
	currentUsername, err := s.usernameFromToken(token)
	if err != nil {
		return model.User{}, err
	}
	if strings.TrimSpace(username) == "" {
		username = currentUsername
	}
	if username != currentUsername {
		if _, err = s.loadUser(username); err == nil {
			return model.User{}, errors.New("username already exists")
		}
		if !errors.Is(err, errUserNotFound) {
			return model.User{}, err
		}
	}
	record, err := s.loadUser(currentUsername)
	if err != nil {
		return model.User{}, err
	}
	record.Username = username
	record.Nickname = nickname
	record.Avatar = avatar
	if strings.TrimSpace(password) != "" {
		hashValue, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if hashErr != nil {
			return model.User{}, hashErr
		}
		record.PasswordHash = string(hashValue)
	}
	record.UpdatedAt = time.Now().Unix()
	if err = s.saveUser(record, currentUsername); err != nil {
		return model.User{}, err
	}
	return toUser(record), nil
}

func (s *APIService) DeleteUserByToken(token string, targetUsername string) error {
	operator, err := s.usernameFromToken(token)
	if err != nil {
		return err
	}
	target := strings.TrimSpace(targetUsername)
	if target == "" {
		target = operator
	}
	if _, err = s.loadUser(target); err != nil {
		return err
	}
	return s.deleteUser(target)
}

func (s *APIService) CreateUser(username, password, nickname, avatar string) (model.User, error) {
	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		return model.User{}, errors.New("username and password required")
	}
	_, err := s.loadUser(username)
	if err == nil {
		return model.User{}, errors.New("username already exists")
	}
	if !errors.Is(err, errUserNotFound) {
		return model.User{}, err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, err
	}
	now := time.Now().Unix()
	record := model.UserRecord{
		Username:     username,
		Nickname:     nickname,
		Avatar:       avatar,
		PasswordHash: string(passwordHash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err = s.saveUser(record, ""); err != nil {
		return model.User{}, err
	}
	return toUser(record), nil
}

func (s *APIService) TokenFromRequest(req *http.Request) (string, error) {
	key := strings.TrimSpace(s.jwtCfg.TokenHeader)
	if key == "" {
		key = "Authorization"
	}
	value := strings.TrimSpace(req.Header.Get(key))
	if value == "" {
		return "", errInvalidToken
	}
	if strings.HasPrefix(strings.ToLower(value), "bearer ") {
		value = strings.TrimSpace(value[7:])
	}
	if value == "" {
		return "", errInvalidToken
	}
	return value, nil
}

// AuthorizeHeaders applies the same JWT and development-mode rules used by
// the Gin API middleware. Connect handlers receive headers and peer metadata
// directly instead of an *http.Request, so this small adapter keeps the
// authentication policy in the shared API service.
func (s *APIService) AuthorizeHeaders(headers http.Header, host string) bool {
	if s == nil || !s.jwtCfg.Enabled {
		return true
	}
	if loader.IsDevMode() {
		return true
	}
	if headers.Get("X-Hamburger-Token") == constant.AppName {
		if strings.EqualFold(strings.TrimSpace(host), "127.0.0.1") || strings.HasPrefix(strings.TrimSpace(host), "127.0.0.1:") {
			return true
		}
	}
	req := &http.Request{Header: headers}
	token, err := s.TokenFromRequest(req)
	if err != nil {
		return false
	}
	_, err = s.validateToken(token)
	return err == nil
}

func (s *APIService) initDefaultUser() error {
	if s.db == nil {
		return nil
	}
	defaultUsername := strings.TrimSpace(s.bboltCfg.DefaultUsername)
	if defaultUsername == "" {
		defaultUsername = "admin"
	}
	defaultPassword := s.bboltCfg.DefaultPassword
	if defaultPassword == "" {
		defaultPassword = "admin"
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(s.userBucket)
		if err != nil {
			return err
		}
		if value := bucket.Get([]byte(defaultUsername)); len(value) > 0 {
			return nil
		}
		passwordHash, hashErr := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if hashErr != nil {
			return hashErr
		}
		now := time.Now().Unix()
		record := model.UserRecord{
			Username:     defaultUsername,
			Nickname:     defaultUsername,
			Avatar:       "",
			PasswordHash: string(passwordHash),
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		payload, marshalErr := json.Marshal(record)
		if marshalErr != nil {
			return marshalErr
		}
		return bucket.Put([]byte(defaultUsername), payload)
	})
}

func (s *APIService) loadUser(username string) (model.UserRecord, error) {
	if s.initErr != nil {
		return model.UserRecord{}, s.initErr
	}
	if s.db == nil {
		return model.UserRecord{}, errStoreUnavailable
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return model.UserRecord{}, errUserNotFound
	}
	var record model.UserRecord
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(s.userBucket)
		if bucket == nil {
			return errUserNotFound
		}
		data := bucket.Get([]byte(username))
		if len(data) == 0 {
			return errUserNotFound
		}
		return json.Unmarshal(data, &record)
	})
	return record, err
}

func (s *APIService) saveUser(record model.UserRecord, oldUsername string) error {
	if s.initErr != nil {
		return s.initErr
	}
	if s.db == nil {
		return errStoreUnavailable
	}
	payload, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket, bucketErr := tx.CreateBucketIfNotExists(s.userBucket)
		if bucketErr != nil {
			return bucketErr
		}
		if oldUsername != "" && oldUsername != record.Username {
			if deleteErr := bucket.Delete([]byte(oldUsername)); deleteErr != nil {
				return deleteErr
			}
		}
		return bucket.Put([]byte(record.Username), payload)
	})
}

func (s *APIService) deleteUser(username string) error {
	if s.initErr != nil {
		return s.initErr
	}
	if s.db == nil {
		return errStoreUnavailable
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return errUserNotFound
	}
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(s.userBucket)
		if bucket == nil {
			return errUserNotFound
		}
		return bucket.Delete([]byte(username))
	})
}

func (s *APIService) createToken(username string) (string, error) {
	secret := []byte(s.jwtCfg.Secret)
	if len(secret) == 0 {
		return "", errors.New("jwt secret is empty")
	}
	method := "HS256"
	if len(s.jwtCfg.AllowedMethods) > 0 {
		for _, item := range s.jwtCfg.AllowedMethods {
			candidate := strings.TrimSpace(item)
			if jwtHashMethod(candidate) != nil {
				method = candidate
				break
			}
		}
	}
	hashFunc := jwtHashMethod(method)
	if hashFunc == nil {
		return "", errors.New("jwt method unsupported")
	}
	header, err := json.Marshal(map[string]interface{}{
		"alg": method,
		"typ": "JWT",
	})
	if err != nil {
		return "", err
	}
	now := time.Now().Unix()
	claims := map[string]interface{}{
		"sub":      username,
		"username": username,
		"iat":      now,
		"nbf":      now,
		"exp":      now + 86400,
	}
	if strings.TrimSpace(s.jwtCfg.Issuer) != "" {
		claims["iss"] = strings.TrimSpace(s.jwtCfg.Issuer)
	}
	if strings.TrimSpace(s.jwtCfg.Audience) != "" {
		claims["aud"] = strings.TrimSpace(s.jwtCfg.Audience)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	part1 := base64.RawURLEncoding.EncodeToString(header)
	part2 := base64.RawURLEncoding.EncodeToString(payload)
	signingInput := part1 + "." + part2
	mac := hmac.New(hashFunc, secret)
	_, _ = mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return signingInput + "." + signature, nil
}

func (s *APIService) usernameFromToken(token string) (string, error) {
	claims, err := s.validateToken(token)
	if err != nil {
		return "", err
	}
	if username, ok := claims["username"].(string); ok && strings.TrimSpace(username) != "" {
		return strings.TrimSpace(username), nil
	}
	if sub, ok := claims["sub"].(string); ok && strings.TrimSpace(sub) != "" {
		return strings.TrimSpace(sub), nil
	}
	return "", errInvalidToken
}

func (s *APIService) validateToken(token string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errInvalidToken
	}
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errInvalidToken
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errInvalidToken
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, errInvalidToken
	}
	var header map[string]interface{}
	if err = json.Unmarshal(headerBytes, &header); err != nil {
		return nil, errInvalidToken
	}
	alg, _ := header["alg"].(string)
	hashFunc := jwtHashMethod(alg)
	if hashFunc == nil || !isAllowedMethod(alg, s.jwtCfg.AllowedMethods) {
		return nil, errInvalidToken
	}
	secret := []byte(s.jwtCfg.Secret)
	if len(secret) == 0 {
		return nil, errInvalidToken
	}
	mac := hmac.New(hashFunc, secret)
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return nil, errInvalidToken
	}
	var claims map[string]interface{}
	if err = json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, errInvalidToken
	}
	now := time.Now().Unix()
	if exp, ok := int64Claim(claims["exp"]); ok && now >= exp {
		return nil, errInvalidToken
	}
	if nbf, ok := int64Claim(claims["nbf"]); ok && now < nbf {
		return nil, errInvalidToken
	}
	if issuer := strings.TrimSpace(s.jwtCfg.Issuer); issuer != "" {
		iss, _ := claims["iss"].(string)
		if iss != issuer {
			return nil, errInvalidToken
		}
	}
	if audience := strings.TrimSpace(s.jwtCfg.Audience); audience != "" && !containsAudience(claims["aud"], audience) {
		return nil, errInvalidToken
	}
	return claims, nil
}

func toUser(record model.UserRecord) model.User {
	return model.User{
		Username: record.Username,
		Nickname: record.Nickname,
		Avatar:   record.Avatar,
	}
}

func int64Claim(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case float64:
		return int64(v), true
	case int64:
		return v, true
	case int:
		return int64(v), true
	default:
		return 0, false
	}
}

func containsAudience(value interface{}, audience string) bool {
	switch aud := value.(type) {
	case string:
		return aud == audience
	case []interface{}:
		for _, item := range aud {
			if str, ok := item.(string); ok && str == audience {
				return true
			}
		}
	}
	return false
}

func jwtHashMethod(method string) func() hash.Hash {
	switch method {
	case "HS256":
		return sha256.New
	case "HS384":
		return sha512.New384
	case "HS512":
		return sha512.New
	default:
		return nil
	}
}

func isAllowedMethod(method string, allowed []string) bool {
	if len(allowed) == 0 {
		return method == "HS256"
	}
	for _, item := range allowed {
		if strings.TrimSpace(item) == method {
			return true
		}
	}
	return false
}
