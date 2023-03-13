package log

import (
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var commitID string

func UserID(userID uuid.UUID) zap.Field {
	return zap.String("user_id", userID.String())
}

type botLogger struct {
	logger *zap.SugaredLogger
}

func NewBotLogger(logger *zap.SugaredLogger) tgbotapi.BotLogger {
	return botLogger{logger: logger}
}

func (b botLogger) Println(v ...interface{}) {
	if len(v) == 1 {
		err, ok := v[0].(error)
		if ok {
			b.logger.Error(err)
			return
		}
	}
	b.logger.Info(v...)
}
func (b botLogger) Printf(format string, v ...interface{}) {
	b.logger.Infof(format, v...)
}

func NewLogger() *zap.SugaredLogger {
	logPath := os.Getenv("LOG_PATH")
	if logPath == "" {
		logPath = "/var/log/bot.log"
	}

	zapConfig := zap.Config{
		Level:       zap.NewAtomicLevelAt(zap.InfoLevel),
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding: "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02T15:04:05.999999Z07:00"),
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{logPath},
		ErrorOutputPaths: []string{logPath},
	}

	l, err := zapConfig.Build()
	if err != nil {
		panic(err)
	}
	return l.Sugar().With(zap.String("commit_id", commitID))
}
