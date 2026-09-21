package trojan

import (
	"Hamburger/exp/trojan/log"
	"Hamburger/internal/config"
	"Hamburger/internal/config/exp_config"
	"Hamburger/internal/logger"
	"context"
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

type TrojanServer struct {
	cfg     *config.Config
	logger  *zerolog.Logger
	cfgPath string
	// 独立的日志记录器
	app *liteTrojan
}

func NewTrojanServer(cfg *config.Config, logger *zerolog.Logger) (*TrojanServer, error) {
	if cfg == nil {
		return nil, nil
	}
	path := strings.TrimSpace(cfg.ExpConfig.TrojanServer)
	server := &TrojanServer{cfg: cfg, logger: logger, cfgPath: path}
	if path == "" || (cfg.ExpConfig.TrojanEnabled != nil && !*cfg.ExpConfig.TrojanEnabled) {
		return server, nil
	}
	ctx := context.Background()
	trojanCfg, err := loadConfig(path)
	if err != nil {
		return nil, err
	}
	buildLog(trojanCfg)
	app, err := newLiteTrojan(ctx, trojanCfg)
	if err != nil {
		return nil, err
	}
	server.app = app
	return server, nil
}

func NewTrojanServerFromConfig(configPath string) (*TrojanServer, error) {
	logger.InitLogger()
	cfg := &config.Config{
		ExpConfig: exp_config.ExpConfig{
			TrojanServer: configPath,
		},
	}
	return NewTrojanServer(cfg, logger.GetLogger())
}

func (t *TrojanServer) Start() error {
	if t == nil || t.app == nil {
		return nil
	}
	return t.app.Run()
}

// StartAsync starts the already-bound Trojan engine in the background. The
// constructor has already opened the listener, so returning means the port
// was bound successfully.
func (t *TrojanServer) StartAsync() error {
	if t == nil || t.app == nil {
		return nil
	}
	go func() { _ = t.app.Run() }()
	return nil
}

// Reload reconstructs the engine from the current main configuration. The
// old engine is closed before binding the new listener; if parsing or binding
// fails, a best-effort instance from the previous file is restored.
func (t *TrojanServer) Reload() error {
	if t == nil || t.cfg == nil {
		return nil
	}
	oldPath := t.cfgPath
	if t.app != nil {
		_ = t.app.Close()
		t.app = nil
	}
	next, err := NewTrojanServer(t.cfg, t.logger)
	if err == nil {
		t.app = next.app
		t.cfgPath = next.cfgPath
		return nil
	}
	if oldPath != "" {
		backupCfg := *t.cfg
		backupCfg.ExpConfig = t.cfg.ExpConfig
		backupCfg.ExpConfig.TrojanServer = oldPath
		if restored, restoreErr := NewTrojanServer(&backupCfg, t.logger); restoreErr == nil && restored != nil {
			t.app = restored.app
			t.cfgPath = oldPath
		}
	}
	return err
}

func (t *TrojanServer) Stop() error {
	if t == nil || t.app == nil {
		return nil
	}
	return t.app.Close()
}

func buildLog(cfg *liteConfig) {
	log.SetLogLevel(log.LogLevel(cfg.Log.LogLevel))
	logFile := strings.TrimSpace(cfg.Log.LogFile)
	disableConsole := cfg.Log.DisableConsole
	var outputs []io.Writer
	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			outputs = append(outputs, file)
		}
	}
	if !disableConsole {
		outputs = append(outputs, os.Stdout)
	}
	if len(outputs) == 0 {
		log.SetOutput(io.Discard)
		return
	}
	if len(outputs) == 1 {
		log.SetOutput(outputs[0])
		return
	}
	log.SetOutput(io.MultiWriter(outputs...))
}
