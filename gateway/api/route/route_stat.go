package route

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (h *Handler) handleStat(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.GetStatCounters())
}

func (h *Handler) handleDomain(c *gin.Context) {
	writeJSONBytes(c, h.service.GetDomainData())
}

func (h *Handler) handleGeo(c *gin.Context) {
	writeJSONBytes(c, h.service.GetGeoData())
}

func (h *Handler) handleConn(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.GetConnData())
}
