package route

import (
	"Hamburger/gateway/api/service"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type managementConfigRequest struct {
	Version string                 `json:"version"`
	Values  map[string]interface{} `json:"values"`
}

type managementDomainStateRequest struct {
	Domain string `json:"domain"`
	State  string `json:"state"`
}

type managementApplyRequest struct {
	Services []string `json:"services"`
}

// RegisterManagement mounts dashboard-only endpoints. The middleware is
// passed in by the API server to keep route and middleware packages acyclic.
func RegisterManagement(engine *gin.Engine, svc *service.APIService, auth gin.HandlerFunc) {
	group := engine.Group("/api/management", auth)
	group.GET("/domains", func(c *gin.Context) { handleManagementDomains(c, svc) })
	group.POST("/domains/state", func(c *gin.Context) { handleManagementDomainState(c, svc) })
	group.GET("/config", func(c *gin.Context) { handleManagementConfigGet(c, svc) })
	group.PUT("/config", func(c *gin.Context) { handleManagementConfigPut(c, svc) })
	group.POST("/config/apply", func(c *gin.Context) { handleManagementConfigApply(c, svc) })
	group.GET("/operations/:id", handleManagementOperation)
}

func handleManagementDomains(c *gin.Context, svc *service.APIService) {
	c.JSON(http.StatusOK, gin.H{"domains": svc.GetManagementDomains()})
}

func handleManagementDomainState(c *gin.Context, svc *service.APIService) {
	var req managementDomainStateRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Domain) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	var err error
	switch strings.ToLower(strings.TrimSpace(req.State)) {
	case "start", "running":
		err = svc.StartDomainService(req.Domain)
	case "stop", "stopped":
		err = svc.StopDomainService(req.Domain)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"message": "state must be start or stop"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func handleManagementConfigGet(c *gin.Context, svc *service.APIService) {
	value, err := svc.GetManagementConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, value)
}

func handleManagementConfigPut(c *gin.Context, svc *service.APIService) {
	var req managementConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Values) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	value, err := svc.SaveManagementConfig(req.Version, req.Values)
	if err != nil {
		code := http.StatusBadRequest
		if strings.Contains(err.Error(), "outside") {
			code = http.StatusConflict
		}
		c.JSON(code, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, value)
}

func handleManagementConfigApply(c *gin.Context, svc *service.APIService) {
	var req managementApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	services := req.Services
	if len(services) == 0 {
		services = []string{"gateway"}
	}
	id := svc.ApplyManagementConfig(services)
	c.JSON(http.StatusAccepted, gin.H{"operation_id": id})
}

func handleManagementOperation(c *gin.Context) {
	value, ok := service.GetManagementOperation(c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"message": "operation not found"})
		return
	}
	value["operation_id"] = c.Param("id")
	c.JSON(http.StatusOK, value)
}
