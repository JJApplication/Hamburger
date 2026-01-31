package modifier

import (
	"Hamburger/internal/config"
	"net/http"
	"strings"
)

type CorsHeaderModifier struct {
	enabled bool
	headers []string
	methods []string
	origins []string
}

var (
	defaultHeaders = []string{"Content-Type", "Origin", "Authorization"}
	defaultMethods = []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE"}
	defaultOrigins = []string{"*"}
)

func NewCorsHeaderModifier() *CorsHeaderModifier {
	cfg := config.Get()
	cm := &CorsHeaderModifier{
		enabled: cfg.Middleware.CORS.Enabled,
		headers: cfg.Middleware.CORS.Header,
		methods: cfg.Middleware.CORS.Method,
		origins: cfg.Middleware.CORS.Origin,
	}

	return cm
}

func (c *CorsHeaderModifier) Use(response *http.Response) {
	c.ModifyResponse(response)
}

func (c *CorsHeaderModifier) ModifyResponse(response *http.Response) error {
	var (
		origins = c.origins
		methods = c.methods
		headers = c.headers
	)

	if len(c.origins) <= 0 {
		origins = defaultOrigins
	}
	if len(c.methods) <= 0 {
		methods = defaultMethods
	}
	if len(headers) <= 0 {
		headers = defaultHeaders
	}
	response.Header.Set("Access-Control-Allow-Origin", strings.Join(origins, ","))
	response.Header.Set("Access-Control-Allow-Methods", strings.Join(methods, ","))
	response.Header.Set("Access-Control-Allow-Headers", strings.Join(headers, ","))
	response.Header.Set("Access-Control-Allow-Credentials", "true")

	return nil
}

func (c *CorsHeaderModifier) IsEnabled() bool {
	return c.enabled
}

func (c *CorsHeaderModifier) UpdateConfig() {
	//TODO implement me
	panic("implement me")
}

func (c *CorsHeaderModifier) GetName() string {
	return "cors"
}
