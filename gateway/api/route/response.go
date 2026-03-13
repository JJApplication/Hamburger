package route

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func writeJSONBytes(c *gin.Context, data []byte) {
	if len(data) == 0 {
		c.Data(http.StatusOK, "application/json", []byte("{}"))
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}
