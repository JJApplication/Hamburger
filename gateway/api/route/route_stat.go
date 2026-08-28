package route

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"

	"Hamburger/gateway/stat"
)

func (h *Handler) handleStat(c *gin.Context) {
	response, err := h.service.GetStatData(c.Query("range"), c.Query("domain"))
	if err != nil {
		if errors.Is(err, stat.ErrInvalidRange) {
			c.JSON(http.StatusBadRequest, gin.H{
				"message":        err.Error(),
				"allowed_ranges": stat.AllowedRanges,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, response)
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
