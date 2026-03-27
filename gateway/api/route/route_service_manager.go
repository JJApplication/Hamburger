package route

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type domainServiceRequest struct {
	Domain string `json:"domain"`
}

type serverServiceRequest struct {
	Server string `json:"server"`
}

func (h *Handler) handleServiceStart(c *gin.Context) {
	var req domainServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	if err := h.service.StartDomainService(strings.TrimSpace(req.Domain)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (h *Handler) handleServiceStop(c *gin.Context) {
	var req domainServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	if err := h.service.StopDomainService(strings.TrimSpace(req.Domain)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (h *Handler) handleServerStop(c *gin.Context) {
	var req serverServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	if err := h.service.StopServer(strings.TrimSpace(req.Server)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

func (h *Handler) handleServerRestart(c *gin.Context) {
	var req serverServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}
	if err := h.service.RestartServer(strings.TrimSpace(req.Server)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}
