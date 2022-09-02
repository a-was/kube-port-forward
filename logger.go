package main

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func customlogTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("15:04:05.000")) //Jan 2
}

func CustomLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("|" + level.CapitalString() + "|")
}

func customCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("\033[38;5;241m" + caller.TrimmedPath() + ":" + "\033[0m")
}

func initLogger() *zap.Logger {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.ConsoleSeparator = " "
	cfg.EncoderConfig.EncodeTime = customlogTimeEncoder
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	// cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	// cfg.EncoderConfig.EncodeCaller = customCallerEncoder
	cfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	os.MkdirAll("/tmp/log/itsy", 0755)
	cfg.OutputPaths = []string{"/tmp/log/itsy/zlog" /* , "stdout" */}
	logger, _ := cfg.Build()
	logger = logger.WithOptions(zap.AddStacktrace(zap.PanicLevel))
	return logger
}
