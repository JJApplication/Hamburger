package route

import (
	"Hamburger/gateway/health_probe"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, health_probe.GetAllProbes())
}
