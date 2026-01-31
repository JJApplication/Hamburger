package logger

import (
	"Hamburger/internal/config"
	"github.com/rs/zerolog"
	"os"
	"sync"
	"time"
)

var (
	globalLogger zerolog.Logger
	once         sync.Once
)

func InitLogger() {
	globalLogger = zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.DateTime,
	}).Level(zerolog.NoLevel).With().Timestamp().Logger()
}

func ReloadLogger(conf *config.LogConfig) {
	once.Do(func() {
		globalLogger = zerolog.New(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			NoColor:    !conf.Color,
			TimeFormat: time.DateTime,
		}).Level(getLevel(conf.LogLevel)).With().Timestamp().Caller().Logger()
	})
}

func L() *zerolog.Logger {
	return &globalLogger
}

func GetLogger() *zerolog.Logger {
	return &globalLogger
}

func Info(msg string) {
	globalLogger.Info().Msg(msg)
}

func Debug(msg string) {
	globalLogger.Debug().Msg(msg)
}

func Warn(msg string) {
	globalLogger.Warn().Msg(msg)
}

func Error(msg string) {
	globalLogger.Error().Msg(msg)
}

//go:inline
func getLevel(level string) zerolog.Level {
	switch level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}
