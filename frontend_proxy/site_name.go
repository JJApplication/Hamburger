package frontend_proxy

import "github.com/gin-gonic/gin"

// resolveSiteName prefers the in-process context value. A header fallback
// remains only for direct local requests to the Helios listener.
func resolveSiteName(c *gin.Context, headerName string) string {
	if c == nil || c.Request == nil {
		return ""
	}
	if name, ok := SiteNameFrom(c.Request.Context()); ok {
		return name
	}
	if headerName == "" {
		return ""
	}
	return c.GetHeader(headerName)
}
