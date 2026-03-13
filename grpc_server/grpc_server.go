package grpc_server

import (
	appgrpc "Hamburger/app/grpc"
	"Hamburger/frontend_proxy"
	"Hamburger/gateway/api"
	"Hamburger/gateway/manager"
	"Hamburger/gateway/modifier"
	"Hamburger/internal/config"
	"net"
	"strings"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

type AppServiceServer struct {
	listener   net.Listener
	grpcServer *grpc.Server
	cfg        *config.Config
	logger     *zerolog.Logger
}

func NewServer(cfg *config.Config, logger *zerolog.Logger,
	getManager func() *manager.Manager,
	getFrontServer func() *frontend_proxy.HeliosServer,
	getModifierManager func() *modifier.ModifierManager,
	getAPIServer func() *api.Server) (*AppServiceServer, error) {
	svr := grpc.NewServer()

	grpcAddr := ""
	if cfg.GRPC.Enabled {
		grpcAddr = strings.TrimSpace(cfg.GRPC.Address)
	}
	network := "tcp"
	if strings.HasSuffix(grpcAddr, ".sock") {
		network = "unix"
	}
	listener, err := net.Listen(network, grpcAddr)
	if err != nil {
		return nil, err
	}
	appgrpc.RegisterAppServiceServer(svr, NewAppService(getManager, getFrontServer, getModifierManager, getAPIServer))

	return &AppServiceServer{
		listener:   listener,
		grpcServer: svr,
		cfg:        cfg,
		logger:     logger,
	}, nil
}

func (s *AppServiceServer) Start() error {
	if !s.cfg.GRPC.Enabled {
		return nil
	}
	if err := s.grpcServer.Serve(s.listener); err != nil {
		return err
	}

	return nil
}

func (s *AppServiceServer) Stop() {
	if !s.cfg.GRPC.Enabled {
		return
	}
	s.grpcServer.GracefulStop()
}
