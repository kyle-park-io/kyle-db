package logger

import (
	"os"
	"strings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *zap.SugaredLogger

// consoleLevel reads LOG_LEVEL. The console core used to be pinned to Debug,
// which is the level the routine paths log at - every ADDUSER, every closed
// connection - so the interesting lines were buried. Info by default; set
// LOG_LEVEL=debug to follow a single session through.
func consoleLevel() zapcore.Level {
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func InitLogger() {
	fileName := viper.GetString("LOG_PATH") + "/logs.log"
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	})

	fileEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	fileCore := zapcore.NewCore(fileEncoder, fileWriter, zapcore.InfoLevel)

	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	consoleWriter := zapcore.AddSync(os.Stdout)
	consoleCore := zapcore.NewCore(consoleEncoder, consoleWriter, consoleLevel())

	core := zapcore.NewTee(fileCore, consoleCore)

	// No `defer logger.Sync()` here: a defer in this function runs when this
	// function returns, which is before a single line has been written. Sync
	// is exported instead, for main to defer.
	Log = zap.New(core).Sugar()
}

// Sync flushes buffered log lines. main defers this.
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
