package route

import (
	connectpb "Hamburger/app/connect"
	"Hamburger/gateway/api/service"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"
)

type Handler struct {
	service *service.APIService
}

// Endpoint describes one built-in API operation. Both the REST and Connect
// registrars consume this catalog so adding an API cannot silently omit one
// of the supported protocols.
type Endpoint struct {
	HTTPMethod      string
	Path            string
	ConnectMethod   string
	RequiresAuth    bool
	RequestAdapter  func() proto.Message
	ResponseAdapter func() proto.Message
}

var endpoints = []Endpoint{
	{http.MethodGet, "/api/stat", "stat", false, func() proto.Message { return new(connectpb.StatRequest) }, func() proto.Message { return new(connectpb.StatResponse) }},
	{http.MethodGet, "/api/geo", "geo", false, func() proto.Message { return new(connectpb.Empty) }, func() proto.Message { return new(connectpb.GeoResponse) }},
	{http.MethodGet, "/api/domain", "domain", false, func() proto.Message { return new(connectpb.Empty) }, func() proto.Message { return new(connectpb.DomainResponse) }},
	{http.MethodGet, "/api/conn", "conn", false, func() proto.Message { return new(connectpb.Empty) }, func() proto.Message { return new(connectpb.ConnResponse) }},
	{http.MethodGet, "/api/health", "health", false, func() proto.Message { return new(connectpb.Empty) }, func() proto.Message { return new(connectpb.HealthResponse) }},
	{http.MethodPost, "/api/login", "login", false, func() proto.Message { return new(connectpb.LoginRequest) }, func() proto.Message { return new(connectpb.LoginResponse) }},
	{http.MethodPost, "/api/logout", "logout", true, func() proto.Message { return new(connectpb.Empty) }, func() proto.Message { return new(connectpb.ActionResponse) }},
	{http.MethodGet, "/api/user", "userGet", true, func() proto.Message { return new(connectpb.Empty) }, func() proto.Message { return new(connectpb.UserResponse) }},
	{http.MethodPut, "/api/user", "userUpdate", true, func() proto.Message { return new(connectpb.UserUpdateRequest) }, func() proto.Message { return new(connectpb.UserResponse) }},
	{http.MethodPost, "/api/user", "userCreate", true, func() proto.Message { return new(connectpb.UserCreateRequest) }, func() proto.Message { return new(connectpb.UserResponse) }},
	{http.MethodDelete, "/api/user", "userDelete", true, func() proto.Message { return new(connectpb.UserDeleteRequest) }, func() proto.Message { return new(connectpb.ActionResponse) }},
	{http.MethodPost, "/api/service/start", "serviceStart", true, func() proto.Message { return new(connectpb.DomainServiceRequest) }, func() proto.Message { return new(connectpb.ActionResponse) }},
	{http.MethodPost, "/api/service/stop", "serviceStop", true, func() proto.Message { return new(connectpb.DomainServiceRequest) }, func() proto.Message { return new(connectpb.ActionResponse) }},
	{http.MethodPost, "/api/server/restart", "serverRestart", true, func() proto.Message { return new(connectpb.ServerRequest) }, func() proto.Message { return new(connectpb.ActionResponse) }},
	{http.MethodPost, "/api/server/stop", "serverStop", true, func() proto.Message { return new(connectpb.ServerRequest) }, func() proto.Message { return new(connectpb.ActionResponse) }},
}

// Endpoints returns a copy of the built-in API catalog.
func Endpoints() []Endpoint {
	return append([]Endpoint(nil), endpoints...)
}

// EndpointForConnectMethod returns the catalog entry used by both protocol
// facades. The returned value is a copy and may be safely modified by a
// caller.
func EndpointForConnectMethod(method string) (Endpoint, bool) {
	for _, endpoint := range endpoints {
		if endpoint.ConnectMethod == method {
			return endpoint, true
		}
	}
	return Endpoint{}, false
}

func Register(engine *gin.Engine, svc *service.APIService, jwt gin.HandlerFunc) {
	h := &Handler{service: svc}
	auth := engine.Group("/api", jwt)
	handlers := map[string]gin.HandlerFunc{
		"stat":          h.handleStat,
		"geo":           h.handleGeo,
		"domain":        h.handleDomain,
		"conn":          h.handleConn,
		"health":        h.handleHealth,
		"login":         h.handleLogin,
		"logout":        h.handleLogout,
		"userGet":       h.handleUserGet,
		"userUpdate":    h.handleUserUpdate,
		"userCreate":    h.handleUserCreate,
		"userDelete":    h.handleUserDelete,
		"serviceStart":  h.handleServiceStart,
		"serviceStop":   h.handleServiceStop,
		"serverRestart": h.handleServerRestart,
		"serverStop":    h.handleServerStop,
	}
	for _, endpoint := range Endpoints() {
		handler, ok := handlers[endpoint.ConnectMethod]
		if !ok {
			panic("API endpoint has no REST handler: " + endpoint.ConnectMethod)
		}
		if endpoint.RequiresAuth {
			auth.Handle(endpoint.HTTPMethod, strings.TrimPrefix(endpoint.Path, "/api"), handler)
		} else {
			engine.Handle(endpoint.HTTPMethod, endpoint.Path, handler)
		}
	}
}
