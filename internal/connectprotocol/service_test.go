package connectprotocol

import (
	connectpb "Hamburger/app/connect"
	"Hamburger/gateway/api/service"
	"Hamburger/gateway/stat"
	"Hamburger/internal/config/svr_config"
	"context"
	"path/filepath"
	"testing"
	"time"

	connectrpc "connectrpc.com/connect"
)

func TestStatResponseConversion(t *testing.T) {
	value := stat.StatResponse{
		Total: 4,
		Summary: stat.StatSummary{
			Status: stat.StatusSummary{Status1xx: 1, Status2xx: 2, Status5xx: 5},
		},
		Connections: map[string]interface{}{
			"gateway": map[string]int64{"active": 3},
		},
	}
	got, err := statResponse(value)
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != value.Total || got.Summary.Status.Status1Xx != 1 || got.Summary.Status.Status2Xx != 2 || got.Summary.Status.Status5Xx != 5 {
		t.Fatalf("unexpected converted response: %+v", got)
	}
	if got.Connections["gateway"].Active != 3 {
		t.Fatalf("unexpected connections: %+v", got.Connections)
	}
}

func TestProtectedUnaryReusesJWTPolicy(t *testing.T) {
	apiService := service.NewAPIService(svr_config.ApiServerConfig{
		JWT:   svr_config.JWTConfig{Enabled: true, Secret: "test-secret", TokenHeader: "Authorization"},
		BBolt: svr_config.APIBBoltConfig{Enabled: true, File: filepath.Join(t.TempDir(), "users.db")},
	})
	defer apiService.CloseDB()
	connectService := NewService(apiService)
	request := connectrpc.NewRequest(&connectpb.Empty{})
	request.Header().Set("Host", "example.com")
	if _, err := connectService.UserGet(context.Background(), request); connectrpc.CodeOf(err) != connectrpc.CodeUnauthenticated {
		t.Fatalf("missing token error = %v, want unauthenticated", err)
	}
	token, _, err := apiService.Login("admin", "admin")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	request.Header().Set("Authorization", "Bearer "+token)
	response, err := connectService.UserGet(context.Background(), request)
	if err != nil {
		t.Fatalf("authenticated user get failed: %v", err)
	}
	if response.Msg.GetUser().GetUsername() != "admin" {
		t.Fatalf("unexpected user response: %v", response.Msg)
	}
}

func TestGatewayStopReturnsBeforeGatewayShutdown(t *testing.T) {
	apiService := service.NewAPIService(svr_config.ApiServerConfig{})
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	apiService.SetServerControl(map[string]func() error{
		"gateway": func() error {
			close(started)
			<-release
			close(done)
			return nil
		},
	}, nil)
	connectService := NewService(apiService)
	request := connectrpc.NewRequest(&connectpb.ServerRequest{Server: "gateway"})
	request.Header().Set("Host", "127.0.0.1")
	response, err := connectService.ServerStop(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetMessage() != "accepted" {
		t.Fatalf("response message = %q, want accepted", response.Msg.GetMessage())
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("gateway stop callback did not start")
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("gateway stop callback did not finish")
	}
}
