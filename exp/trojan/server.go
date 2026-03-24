package trojan

import (
	"Hamburger/exp/trojan/log"
	"Hamburger/internal/config"
	"Hamburger/internal/logger"
	"context"
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

type TrojanServer struct {
	cfg    *config.Config
	logger *zerolog.Logger
	// 独立的日志记录器
	app *liteTrojan
}

func NewTrojanServer(cfg *config.Config, logger *zerolog.Logger) (*TrojanServer, error) {
	if cfg == nil || cfg.ExpConfig.TrojanServer == "" {
		return nil, nil
	}
	ctx := context.Background()
	trojanCfg, err := loadConfig(cfg.ExpConfig.TrojanServer)
	if err != nil {
		return nil, err
	}
	buildLog(trojanCfg)
	app, err := newLiteTrojan(ctx, trojanCfg)
	if err != nil {
		return nil, err
	}
	return &TrojanServer{
		cfg:    cfg,
		logger: logger,
		app:    app,
	}, nil
}

func NewTrojanServerFromConfig(configPath string) (*TrojanServer, error) {
	logger.InitLogger()
	cfg := &config.Config{
		ExpConfig: config.ExpConfig{
			TrojanServer: configPath,
		},
	}
	return NewTrojanServer(cfg, logger.GetLogger())
}

func (t *TrojanServer) Start() error {
	if t.app == nil {
		return nil
	}
	return t.app.Run()
}

func (t *TrojanServer) Stop() error {
	if t.app == nil {
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
