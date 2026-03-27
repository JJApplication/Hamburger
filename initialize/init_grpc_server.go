package initialize

import (
	"Hamburger/backend_proxy"
	"Hamburger/exp/any_tls"
	"Hamburger/exp/trojan"
	"Hamburger/exp/vpn_proxy"
	"Hamburger/frontend_proxy"
	"Hamburger/gateway/api"
	"Hamburger/gateway/latency"
	"Hamburger/gateway/manager"
	"Hamburger/gateway/modifier"
	"Hamburger/grpc_server"
	"Hamburger/internal/config"
	"errors"
)

func (i *Initializer) InitGRPCServer() Runner {
	return Runner{
		Priority: PriorityFinal,
		fn: func() error {
			i.bindAPIServerControl()
			gs, err := grpc_server.NewServer(i.cfg, i.logger,
				func() *manager.Manager { return i.Manager },
				func() *frontend_proxy.HeliosServer { return i.FrontServer },
				func() *modifier.ModifierManager { return i.ModifierManager },
				func() *api.Server { return i.APIServer },
				func() *backend_proxy.BackendProxy { return i.BackendServer },
				func() *latency.LatencyServer { return i.LatencyServer },
				func() *vpn_proxy.VpnServer { return i.VpnServer },
				func() *trojan.TrojanServer { return i.TrojanServer },
				func() *any_tls.AnyTLSServer { return i.AnyTLSServer })
			if err != nil {
				return err
			}
			i.GrpcServer = gs
			i.logger.Info().Msg("grpc server initialized")
			return nil
		},
	}
}

func (i *Initializer) bindAPIServerControl() {
	if i.APIServer == nil {
		return
	}
	stopFn := map[string]func() error{
		"gateway": func() error {
			if i.Manager == nil {
				return errors.New("gateway manager unavailable")
			}
			return i.Manager.Stop()
		},
		"front": func() error {
			if i.FrontServer == nil {
				return errors.New("front server unavailable")
			}
			i.FrontServer.Shutdown()
			return nil
		},
		"api": func() error {
			go func() {
				_ = i.APIServer.Stop()
			}()
			return nil
		},
		"backend": func() error {
			if i.BackendServer == nil {
				return errors.New("backend server unavailable")
			}
			i.BackendServer.Stop()
			return nil
		},
		"latency": func() error {
			if i.LatencyServer == nil {
				return errors.New("latency server unavailable")
			}
			return i.LatencyServer.Stop()
		},
		"vpn": func() error {
			if i.VpnServer == nil {
				return errors.New("vpn server unavailable")
			}
			return i.VpnServer.Stop()
		},
		"trojan": func() error {
			if i.TrojanServer == nil {
				return errors.New("trojan server unavailable")
			}
			return i.TrojanServer.Stop()
		},
		"anytls": func() error {
			if i.AnyTLSServer == nil {
				return errors.New("anytls server unavailable")
			}
			return i.AnyTLSServer.Stop()
		},
	}
	restartFn := map[string]func() error{
		"gateway": func() error {
			if i.Manager == nil {
				return errors.New("gateway manager unavailable")
			}
			if err := reloadConfigInPlace(i.cfg); err != nil {
				return err
			}
			return i.Manager.Restart()
		},
		"front": func() error {
			if i.FrontServer == nil {
				return errors.New("front server unavailable")
			}
			if err := reloadConfigInPlace(i.cfg); err != nil {
				return err
			}
			i.FrontServer.Shutdown()
			go func() {
				_ = i.FrontServer.Start()
			}()
			return nil
		},
		"api": func() error {
			if i.APIServer == nil {
				return errors.New("api server unavailable")
			}
			if err := reloadConfigInPlace(i.cfg); err != nil {
				return err
			}
			go func() {
				_ = i.APIServer.Stop()
				_ = i.APIServer.Start()
			}()
			return nil
		},
		"backend": func() error {
			if i.BackendServer == nil {
				return errors.New("backend server unavailable")
			}
			if err := reloadConfigInPlace(i.cfg); err != nil {
				return err
			}
			i.BackendServer.Stop()
			i.BackendServer.Start()
			return nil
		},
		"latency": func() error {
			if i.LatencyServer == nil {
				return errors.New("latency server unavailable")
			}
			if err := reloadConfigInPlace(i.cfg); err != nil {
				return err
			}
			if err := i.LatencyServer.Stop(); err != nil {
				return err
			}
			return i.LatencyServer.Start()
		},
		"vpn": func() error {
			if i.VpnServer == nil {
				return errors.New("vpn server unavailable")
			}
			if err := reloadConfigInPlace(i.cfg); err != nil {
				return err
			}
			if err := i.VpnServer.Stop(); err != nil {
				return err
			}
			return i.VpnServer.Start()
		},
		"trojan": func() error {
			if i.TrojanServer == nil {
				return errors.New("trojan server unavailable")
			}
			if err := reloadConfigInPlace(i.cfg); err != nil {
				return err
			}
			if err := i.TrojanServer.Stop(); err != nil {
				return err
			}
			return i.TrojanServer.Start()
		},
		"anytls": func() error {
			if i.AnyTLSServer == nil {
				return errors.New("anytls server unavailable")
			}
			if err := reloadConfigInPlace(i.cfg); err != nil {
				return err
			}
			if err := i.AnyTLSServer.Stop(); err != nil {
				return err
			}
			return i.AnyTLSServer.Start()
		},
	}
	i.APIServer.SetServerControl(stopFn, restartFn)
}

func reloadConfigInPlace(currentCfg *config.Config) error {
	appCfg, err := config.LoadConfig("config/config.json")
	if err != nil {
		return err
	}
	mergedCfg := config.Merge(appCfg)
	if currentCfg == nil {
		config.Set(mergedCfg)
		return nil
	}
	*currentCfg = *mergedCfg
	config.Set(currentCfg)
	return nil
}
