package utils

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

var (
	log         *LoggerService
	initOnce    sync.Once
)

type Logger interface {
	Info(msg string)
	Warn(msg string)
	Error(msg string, err error)
	Fatal(msg string, err error)
}

type LoggerService struct {
	log zerolog.Logger
}

func NewLoggerService() *LoggerService {
	var output io.Writer = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}

	logger := zerolog.New(output).With().Timestamp().Logger()
	return &LoggerService{
		log: logger,
	}
}

func InitLoggerOnce() {
	initOnce.Do(func() {
		log = NewLoggerService()
		log.Info("[LOG]: Logger initialized successfully")
	})
}

func GetLogger() *LoggerService {
	if log == nil {
		InitLoggerOnce()
	}
	return log
}

func (l *LoggerService) Info(msg string) {
	l.log.WithLevel(zerolog.InfoLevel).Msgf("%s", msg)
}

func (l *LoggerService) Warn(msg string) {
	l.log.WithLevel(zerolog.WarnLevel).Msgf("%s", msg)
}

func (l *LoggerService) Error(msg string, err error) {
	l.log.WithLevel(zerolog.ErrorLevel).Err(err).Msgf("%s", msg)
}

// Fatal Logs can be used to log fatal errors and it exits the application
// It is recommended to use this only for critical errors that should stop the application
func (l *LoggerService) Fatal(msg string, err error) {
	l.log.WithLevel(zerolog.FatalLevel).Err(err).Msgf("%s", msg)
}