package route

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type updateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

type deleteUserRequest struct {
	Username string `json:"username"`
}

func (h *Handler) handleLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	token, user, err := h.service.Login(strings.TrimSpace(req.Username), req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid username or password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  user,
	})
}

func (h *Handler) handleLogout(c *gin.Context) {
	if err := h.service.Logout(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (h *Handler) handleUserGet(c *gin.Context) {
	token, err := h.service.TokenFromRequest(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	user, err := h.service.GetUserByToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *Handler) handleUserUpdate(c *gin.Context) {
	token, err := h.service.TokenFromRequest(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	var req updateUserRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	user, err := h.service.UpdateUserByToken(token, strings.TrimSpace(req.Username), strings.TrimSpace(req.Nickname), strings.TrimSpace(req.Avatar), req.Password)
	if err != nil {
		code := http.StatusInternalServerError
		if strings.Contains(err.Error(), "exists") {
			code = http.StatusConflict
		}
		if strings.Contains(err.Error(), "token") {
			code = http.StatusUnauthorized
		}
		c.JSON(code, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *Handler) handleUserCreate(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	token, err := h.service.TokenFromRequest(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	if _, err = h.service.GetUserByToken(token); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	user, err := h.service.CreateUser(strings.TrimSpace(req.Username), req.Password, strings.TrimSpace(req.Nickname), strings.TrimSpace(req.Avatar))
	if err != nil {
		code := http.StatusInternalServerError
		if strings.Contains(err.Error(), "required") {
			code = http.StatusBadRequest
		}
		if strings.Contains(err.Error(), "exists") {
			code = http.StatusConflict
		}
		c.JSON(code, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *Handler) handleUserDelete(c *gin.Context) {
	token, err := h.service.TokenFromRequest(c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}
	var req deleteUserRequest
	if err = c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	if err = h.service.DeleteUserByToken(token, strings.TrimSpace(req.Username)); err != nil {
		code := http.StatusInternalServerError
		if strings.Contains(err.Error(), "token") {
			code = http.StatusUnauthorized
		}
		c.JSON(code, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
