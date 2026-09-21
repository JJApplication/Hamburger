package middleware

import (
	"Hamburger/gateway/api/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ManagementJWT never inherits the public API's dev-mode bypass. Dashboard
// management endpoints must prove possession of a valid user token.
func ManagementJWT(api *service.APIService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if api == nil || !api.AuthorizeManagementHeaders(c.Request.Header) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}
		c.Next()
	}
}
