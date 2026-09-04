package connectprotocol

import (
	connectpb "Hamburger/app/connect"
	"Hamburger/gateway/api/route"
	"Hamburger/gateway/api/service"
	"Hamburger/gateway/health_probe"
	"Hamburger/gateway/stat"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	connectrpc "connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
)

// Service implements the generated hamburger.service Connect contract on top
// of the same APIService used by the existing Gin routes.
type Service struct {
	api *service.APIService
}

func NewService(apiService *service.APIService) *Service {
	return &Service{api: apiService}
}

func (s *Service) Stat(ctx context.Context, req *connectrpc.Request[connectpb.StatRequest]) (*connectrpc.Response[connectpb.StatResponse], error) {
	if err := s.authorize("stat", req.Header(), req.Peer()); err != nil {
		return nil, err
	}
	if s.api == nil {
		return nil, unavailable("api service unavailable")
	}
	rangeValue := req.Msg.GetRange()
	domain := req.Msg.GetDomain()
	// Accept the REST query names as a compatibility convenience when a
	// caller sends an otherwise empty Connect request.
	if req.Peer().Query != nil {
		if rangeValue == "" {
			rangeValue = req.Peer().Query.Get("range")
		}
		if domain == "" {
			domain = req.Peer().Query.Get("domain")
		}
	}
	value, err := s.api.GetStatData(rangeValue, domain)
	if err != nil {
		return nil, rpcError(err)
	}
	response, err := statResponse(value)
	if err != nil {
		return nil, fmt.Errorf("encode stat response: %w", err)
	}
	return connectrpc.NewResponse(response), nil
}

func (s *Service) Geo(_ context.Context, req *connectrpc.Request[connectpb.Empty]) (*connectrpc.Response[connectpb.GeoResponse], error) {
	if err := s.authorize("geo", req.Header(), req.Peer()); err != nil {
		return nil, err
	}
	if s.api == nil {
		return nil, unavailable("api service unavailable")
	}
	values, err := int64Map(s.api.GetGeoData())
	if err != nil {
		return nil, fmt.Errorf("decode geo response: %w", err)
	}
	return connectrpc.NewResponse(&connectpb.GeoResponse{Values: values}), nil
}

func (s *Service) Domain(_ context.Context, req *connectrpc.Request[connectpb.Empty]) (*connectrpc.Response[connectpb.DomainResponse], error) {
	if err := s.authorize("domain", req.Header(), req.Peer()); err != nil {
		return nil, err
	}
	if s.api == nil {
		return nil, unavailable("api service unavailable")
	}
	values, err := int64Map(s.api.GetDomainData())
	if err != nil {
		return nil, fmt.Errorf("decode domain response: %w", err)
	}
	return connectrpc.NewResponse(&connectpb.DomainResponse{Values: values}), nil
}

func (s *Service) Conn(_ context.Context, req *connectrpc.Request[connectpb.Empty]) (*connectrpc.Response[connectpb.ConnResponse], error) {
	if err := s.authorize("conn", req.Header(), req.Peer()); err != nil {
		return nil, err
	}
	return connectrpc.NewResponse(&connectpb.ConnResponse{Values: connectionValues()}), nil
}

func (s *Service) Health(_ context.Context, req *connectrpc.Request[connectpb.Empty]) (*connectrpc.Response[connectpb.HealthResponse], error) {
	if err := s.authorize("health", req.Header(), req.Peer()); err != nil {
		return nil, err
	}
	return connectrpc.NewResponse(&connectpb.HealthResponse{Values: health_probe.GetAllProbes()}), nil
}

func (s *Service) Login(_ context.Context, req *connectrpc.Request[connectpb.LoginRequest]) (*connectrpc.Response[connectpb.LoginResponse], error) {
	if err := s.authorize("login", req.Header(), req.Peer()); err != nil {
		return nil, err
	}
	if s.api == nil {
		return nil, unavailable("api service unavailable")
	}
	token, user, err := s.api.Login(strings.TrimSpace(req.Msg.GetUsername()), req.Msg.GetPassword())
	if err != nil {
		return nil, rpcError(err)
	}
	return connectrpc.NewResponse(&connectpb.LoginResponse{
		Token: token,
		User:  &connectpb.User{Username: user.Username, Nickname: user.Nickname, Avatar: user.Avatar},
	}), nil
}

func (s *Service) Logout(_ context.Context, req *connectrpc.Request[connectpb.Empty]) (*connectrpc.Response[connectpb.ActionResponse], error) {
	if err := s.authorize("logout", req.Header(), req.Peer()); err != nil {
		return nil, err
	}
	if s.api == nil {
		return nil, unavailable("api service unavailable")
	}
	if err := s.api.Logout(); err != nil {
		return nil, rpcError(err)
	}
	return connectrpc.NewResponse(okAction()), nil
}

func (s *Service) UserGet(_ context.Context, req *connectrpc.Request[connectpb.Empty]) (*connectrpc.Response[connectpb.UserResponse], error) {
	token, err := s.authToken("userGet", req.Header(), req.Peer())
	if err != nil {
		return nil, err
	}
	user, err := s.api.GetUserByToken(token)
	if err != nil {
		return nil, rpcError(err)
	}
	return connectrpc.NewResponse(&connectpb.UserResponse{User: toProtoUser(user.Username, user.Nickname, user.Avatar)}), nil
}

func (s *Service) UserUpdate(_ context.Context, req *connectrpc.Request[connectpb.UserUpdateRequest]) (*connectrpc.Response[connectpb.UserResponse], error) {
	token, err := s.authToken("userUpdate", req.Header(), req.Peer())
	if err != nil {
		return nil, err
	}
	user, err := s.api.UpdateUserByToken(token, strings.TrimSpace(req.Msg.GetUsername()), strings.TrimSpace(req.Msg.GetNickname()), strings.TrimSpace(req.Msg.GetAvatar()), req.Msg.GetPassword())
	if err != nil {
		return nil, rpcError(err)
	}
	return connectrpc.NewResponse(&connectpb.UserResponse{User: toProtoUser(user.Username, user.Nickname, user.Avatar)}), nil
}

func (s *Service) UserCreate(_ context.Context, req *connectrpc.Request[connectpb.UserCreateRequest]) (*connectrpc.Response[connectpb.UserResponse], error) {
	token, err := s.authToken("userCreate", req.Header(), req.Peer())
	if err != nil {
		return nil, err
	}
	if _, err := s.api.GetUserByToken(token); err != nil {
		return nil, rpcError(err)
	}
	user, err := s.api.CreateUser(strings.TrimSpace(req.Msg.GetUsername()), req.Msg.GetPassword(), strings.TrimSpace(req.Msg.GetNickname()), strings.TrimSpace(req.Msg.GetAvatar()))
	if err != nil {
		return nil, rpcError(err)
	}
	return connectrpc.NewResponse(&connectpb.UserResponse{User: toProtoUser(user.Username, user.Nickname, user.Avatar)}), nil
}

func (s *Service) UserDelete(_ context.Context, req *connectrpc.Request[connectpb.UserDeleteRequest]) (*connectrpc.Response[connectpb.ActionResponse], error) {
	token, err := s.authToken("userDelete", req.Header(), req.Peer())
	if err != nil {
		return nil, err
	}
	if err := s.api.DeleteUserByToken(token, strings.TrimSpace(req.Msg.GetUsername())); err != nil {
		return nil, rpcError(err)
	}
	return connectrpc.NewResponse(okAction()), nil
}

func (s *Service) ServiceStart(_ context.Context, req *connectrpc.Request[connectpb.DomainServiceRequest]) (*connectrpc.Response[connectpb.ActionResponse], error) {
	if err := s.authorize("serviceStart", req.Header(), req.Peer()); err != nil {
		return nil, err
	}
	if err := s.api.StartDomainService(strings.TrimSpace(req.Msg.GetDomain())); err != nil {
		return nil, rpcError(err)
	}
	return connectrpc.NewResponse(okAction()), nil
}

func (s *Service) ServiceStop(_ context.Context, req *connectrpc.Request[connectpb.DomainServiceRequest]) (*connectrpc.Response[connectpb.ActionResponse], error) {
	if err := s.authorize("serviceStop", req.Header(), req.Peer()); err != nil {
		return nil, err
	}
	if err := s.api.StopDomainService(strings.TrimSpace(req.Msg.GetDomain())); err != nil {
		return nil, rpcError(err)
	}
	return connectrpc.NewResponse(okAction()), nil
}

func (s *Service) ServerRestart(_ context.Context, req *connectrpc.Request[connectpb.ServerRequest]) (*connectrpc.Response[connectpb.ActionResponse], error) {
	if err := s.authorize("serverRestart", req.Header(), req.Peer()); err != nil {
		return nil, err
	}
	server := strings.TrimSpace(req.Msg.GetServer())
	var err error
	if strings.EqualFold(server, "gateway") {
		err = s.api.RestartServerAsync(server)
	} else {
		err = s.api.RestartServer(server)
	}
	if err != nil {
		return nil, rpcError(err)
	}
	if strings.EqualFold(server, "gateway") {
		return connectrpc.NewResponse(acceptedAction()), nil
	}
	return connectrpc.NewResponse(okAction()), nil
}

func (s *Service) ServerStop(_ context.Context, req *connectrpc.Request[connectpb.ServerRequest]) (*connectrpc.Response[connectpb.ActionResponse], error) {
	if err := s.authorize("serverStop", req.Header(), req.Peer()); err != nil {
		return nil, err
	}
	server := strings.TrimSpace(req.Msg.GetServer())
	var err error
	if strings.EqualFold(server, "gateway") {
		err = s.api.StopServerAsync(server)
	} else {
		err = s.api.StopServer(server)
	}
	if err != nil {
		return nil, rpcError(err)
	}
	if strings.EqualFold(server, "gateway") {
		return connectrpc.NewResponse(acceptedAction()), nil
	}
	return connectrpc.NewResponse(okAction()), nil
}

func (s *Service) StatStream(ctx context.Context, stream *connectrpc.BidiStream[connectpb.StatRequest, connectpb.StatResponse]) error {
	return serveBidi(ctx, stream, s.Stat)
}

func (s *Service) GeoStream(ctx context.Context, stream *connectrpc.BidiStream[connectpb.Empty, connectpb.GeoResponse]) error {
	return serveBidi(ctx, stream, s.Geo)
}

func (s *Service) DomainStream(ctx context.Context, stream *connectrpc.BidiStream[connectpb.Empty, connectpb.DomainResponse]) error {
	return serveBidi(ctx, stream, s.Domain)
}

func (s *Service) ConnStream(ctx context.Context, stream *connectrpc.BidiStream[connectpb.Empty, connectpb.ConnResponse]) error {
	return serveBidi(ctx, stream, s.Conn)
}

func (s *Service) HealthStream(ctx context.Context, stream *connectrpc.BidiStream[connectpb.Empty, connectpb.HealthResponse]) error {
	return serveBidi(ctx, stream, s.Health)
}

func (s *Service) LoginStream(ctx context.Context, stream *connectrpc.BidiStream[connectpb.LoginRequest, connectpb.LoginResponse]) error {
	return serveBidi(ctx, stream, s.Login)
}

func (s *Service) LogoutStream(ctx context.Context, stream *connectrpc.BidiStream[connectpb.Empty, connectpb.ActionResponse]) error {
	if err := s.authorize("logout", stream.RequestHeader(), stream.Peer()); err != nil {
		return err
	}
	return serveBidi(ctx, stream, s.Logout)
}

func (s *Service) UserGetStream(ctx context.Context, stream *connectrpc.BidiStream[connectpb.Empty, connectpb.UserResponse]) error {
	if err := s.authorize("userGet", stream.RequestHeader(), stream.Peer()); err != nil {
		return err
	}
	return serveBidi(ctx, stream, s.UserGet)
}

func (s *Service) UserUpdateStream(ctx context.Context, stream *connectrpc.BidiStream[connectpb.UserUpdateRequest, connectpb.UserResponse]) error {
	if err := s.authorize("userUpdate", stream.RequestHeader(), stream.Peer()); err != nil {
		return err
	}
	return serveBidi(ctx, stream, s.UserUpdate)
}

func (s *Service) UserCreateStream(ctx context.Context, stream *connectrpc.BidiStream[connectpb.UserCreateRequest, connectpb.UserResponse]) error {
	if err := s.authorize("userCreate", stream.RequestHeader(), stream.Peer()); err != nil {
		return err
	}
	return serveBidi(ctx, stream, s.UserCreate)
}

func (s *Service) UserDeleteStream(ctx context.Context, stream *connectrpc.BidiStream[connectpb.UserDeleteRequest, connectpb.ActionResponse]) error {
	if err := s.authorize("userDelete", stream.RequestHeader(), stream.Peer()); err != nil {
		return err
	}
	return serveBidi(ctx, stream, s.UserDelete)
}

func (s *Service) ServiceStartStream(ctx context.Context, stream *connectrpc.BidiStream[connectpb.DomainServiceRequest, connectpb.ActionResponse]) error {
	if err := s.authorize("serviceStart", stream.RequestHeader(), stream.Peer()); err != nil {
		return err
	}
	return serveBidi(ctx, stream, s.ServiceStart)
}

func (s *Service) ServiceStopStream(ctx context.Context, stream *connectrpc.BidiStream[connectpb.DomainServiceRequest, connectpb.ActionResponse]) error {
	if err := s.authorize("serviceStop", stream.RequestHeader(), stream.Peer()); err != nil {
		return err
	}
	return serveBidi(ctx, stream, s.ServiceStop)
}

func (s *Service) ServerRestartStream(ctx context.Context, stream *connectrpc.BidiStream[connectpb.ServerRequest, connectpb.ActionResponse]) error {
	if err := s.authorize("serverRestart", stream.RequestHeader(), stream.Peer()); err != nil {
		return err
	}
	return serveBidi(ctx, stream, s.ServerRestart)
}

func (s *Service) ServerStopStream(ctx context.Context, stream *connectrpc.BidiStream[connectpb.ServerRequest, connectpb.ActionResponse]) error {
	if err := s.authorize("serverStop", stream.RequestHeader(), stream.Peer()); err != nil {
		return err
	}
	return serveBidi(ctx, stream, s.ServerStop)
}

type unaryHandler[Req any, Res any] func(context.Context, *connectrpc.Request[Req]) (*connectrpc.Response[Res], error)

func serveBidi[Req any, Res any](ctx context.Context, stream *connectrpc.BidiStream[Req, Res], handler unaryHandler[Req, Res]) error {
	for {
		request, err := stream.Receive()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		wrapped := connectrpc.NewRequest(request)
		requestHeaders := stream.RequestHeader()
		copyHeaders(wrapped.Header(), requestHeaders)
		if host := strings.TrimSpace(requestHeaders.Get("Host")); host != "" {
			wrapped.Header().Set("Host", host)
		} else if peer := stream.Peer(); peer.Addr != "" {
			wrapped.Header().Set("Host", peer.Addr)
		}
		response, err := handler(ctx, wrapped)
		if err != nil {
			return err
		}
		if err := stream.Send(response.Msg); err != nil {
			return err
		}
	}
}

func (s *Service) authorize(method string, headers http.Header, peer connectrpc.Peer) error {
	endpoint, ok := route.EndpointForConnectMethod(method)
	if !ok {
		return connectrpc.NewError(connectrpc.CodeUnimplemented, fmt.Errorf("Connect method %q is not registered", method))
	}
	if !endpoint.RequiresAuth {
		return nil
	}
	return s.requireAuth(headers, peer)
}

func (s *Service) requireAuth(headers http.Header, peer connectrpc.Peer) error {
	if s.api == nil {
		return unavailable("api service unavailable")
	}
	host := strings.TrimSpace(headers.Get("Host"))
	if host == "" {
		host = peer.Addr
	}
	if !s.api.AuthorizeHeaders(headers, host) {
		return unauthenticated("unauthorized")
	}
	return nil
}

func (s *Service) authToken(method string, headers http.Header, peer connectrpc.Peer) (string, error) {
	if err := s.authorize(method, headers, peer); err != nil {
		return "", err
	}
	if s.api == nil {
		return "", unavailable("api service unavailable")
	}
	request := &http.Request{Header: headers}
	token, err := s.api.TokenFromRequest(request)
	if err != nil {
		// Authentication-disabled and development-mode requests still need a
		// token for operations whose business API is token-based. This mirrors
		// the existing API handlers, which obtain the token before the call.
		return "", unauthenticated("unauthorized")
	}
	return token, nil
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func statResponse(value stat.StatResponse) (*connectpb.StatResponse, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	response := new(connectpb.StatResponse)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, response); err != nil {
		return nil, err
	}
	return response, nil
}

func int64Map(data []byte) (map[string]int64, error) {
	values := map[string]int64{}
	if len(data) == 0 {
		return values, nil
	}
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	if values == nil {
		values = map[string]int64{}
	}
	return values, nil
}

func connectionValues() map[string]*connectpb.ConnectionState {
	gateway := stat.GetGatewayConn()
	front := stat.GetFrontConn()
	return map[string]*connectpb.ConnectionState{
		"gateway": toConnectionState(gateway.New, gateway.Active, gateway.Idle, gateway.Hijacked, gateway.Closed),
		"front":   toConnectionState(front.New, front.Active, front.Idle, front.Hijacked, front.Closed),
	}
}

func toConnectionState(newCount, active, idle, hijacked, closed int64) *connectpb.ConnectionState {
	return &connectpb.ConnectionState{New: newCount, Active: active, Idle: idle, Hijacked: hijacked, Closed: closed}
}

func toProtoUser(username, nickname, avatar string) *connectpb.User {
	return &connectpb.User{Username: username, Nickname: nickname, Avatar: avatar}
}

func okAction() *connectpb.ActionResponse {
	return &connectpb.ActionResponse{Success: true, Message: "ok"}
}

func acceptedAction() *connectpb.ActionResponse {
	return &connectpb.ActionResponse{Success: true, Message: "accepted"}
}

func unavailable(message string) error {
	return connectrpc.NewError(connectrpc.CodeUnavailable, errors.New(message))
}

func unauthenticated(message string) error {
	return connectrpc.NewError(connectrpc.CodeUnauthenticated, errors.New(message))
}

func rpcError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return connectrpc.NewError(connectrpc.CodeCanceled, err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return connectrpc.NewError(connectrpc.CodeDeadlineExceeded, err)
	}
	message := err.Error()
	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "invalid username"), strings.Contains(lower, "invalid password"), strings.Contains(lower, "invalid token"), strings.Contains(lower, "unauthorized"), strings.Contains(lower, "token"), strings.Contains(lower, "user not found"):
		return connectrpc.NewError(connectrpc.CodeUnauthenticated, err)
	case strings.Contains(lower, "required"), strings.Contains(lower, "empty"), strings.Contains(lower, "invalid"), strings.Contains(lower, "unsupported"):
		return connectrpc.NewError(connectrpc.CodeInvalidArgument, err)
	case strings.Contains(lower, "already exists"), strings.Contains(lower, "exists"):
		return connectrpc.NewError(connectrpc.CodeAlreadyExists, err)
	case strings.Contains(lower, "not found"):
		return connectrpc.NewError(connectrpc.CodeNotFound, err)
	case strings.Contains(lower, "unavailable"):
		return connectrpc.NewError(connectrpc.CodeUnavailable, err)
	default:
		return connectrpc.NewError(connectrpc.CodeInternal, err)
	}
}

var _ connectpb.ServiceHandler = (*Service)(nil)
