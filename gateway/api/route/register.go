package route

import (
	"Hamburger/gateway/api/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *service.APIService
}

func Register(engine *gin.Engine, svc *service.APIService, jwt gin.HandlerFunc) {
	h := &Handler{service: svc}
	engine.GET("/api/stat", h.handleStat)
	engine.GET("/api/geo", h.handleGeo)
	engine.GET("/api/domain", h.handleDomain)
	engine.GET("/api/conn", h.handleConn)
	engine.GET("/api/health", h.handleHealth)
	engine.POST("/api/login", h.handleLogin)

	auth := engine.Group("/api", jwt)
	auth.POST("/logout", h.handleLogout)
	auth.GET("/user", h.handleUserGet)
	auth.PUT("/user", h.handleUserUpdate)
	auth.POST("/user", h.handleUserCreate)
	auth.DELETE("/user", h.handleUserDelete)
}
