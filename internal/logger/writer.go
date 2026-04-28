package logger

import (
	"Hamburger/internal/config/core_config"
	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
)

func NewWriter(cfg *core_config.LogConfig) io.Writer {
	if cfg.LogFile != "" {
		rotator := &lumberjack.Logger{
			Filename: cfg.LogFile,
			MaxSize:  10 * 1 << 20,
			Compress: true,
		}
		fileWriter := io.MultiWriter(zerolog.SyncWriter(os.Stdout), zerolog.SyncWriter(rotator))
		return fileWriter
	}
	return os.Stdout
}
