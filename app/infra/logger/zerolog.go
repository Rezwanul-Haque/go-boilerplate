package logger

import (
	"os"

	"github.com/rs/zerolog"
	"go-boilerplate/app/shared/config"
	"go-boilerplate/app/shared/ports"
)

type zerologLogger struct {
	log zerolog.Logger
}

func New(cfg *config.Config) ports.Logger {
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	log := zerolog.New(os.Stdout).With().Timestamp().Logger()
	return &zerologLogger{log: log}
}

func (l *zerologLogger) Info(msg string, fields ...any) {
	l.log.Info().Fields(fields).Msg(msg)
}

func (l *zerologLogger) Error(msg string, err error, fields ...any) {
	l.log.Error().Err(err).Fields(fields).Msg(msg)
}

func (l *zerologLogger) Debug(msg string, fields ...any) {
	l.log.Debug().Fields(fields).Msg(msg)
}

func (l *zerologLogger) Warn(msg string, fields ...any) {
	l.log.Warn().Fields(fields).Msg(msg)
}
