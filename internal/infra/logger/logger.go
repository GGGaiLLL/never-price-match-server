package logger

import (
	"os"
	"time"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var L *zap.Logger

func Init(env string) {
	enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:       "ts",
		LevelKey:      "level",
		MessageKey:    "msg",
		CallerKey:     "caller",
		EncodeTime:    zapcore.TimeEncoderOfLayout(time.RFC3339),
		EncodeLevel:   zapcore.LowercaseLevelEncoder,
		EncodeCaller:  zapcore.ShortCallerEncoder,
	})
	level := zapcore.InfoLevel
	ws := zapcore.AddSync(os.Stdout)

	core := zapcore.NewCore(enc, ws, level)
	opts := []zap.Option{zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)}
	if env == "dev" {
		opts = append(opts, zap.Development())
	}
	L = zap.New(core, opts...)
}

func Sync() { _ = L.Sync() }


func Field(k string, v any) zap.Field { return zap.Any(k, v) }
func Err(err error) zap.Field                 { return zap.Error(err) }
func Str(k, v string) zap.Field               { return zap.String(k, v) }
func Int(k string, v int) zap.Field           { return zap.Int(k, v) }
func Dur(k string, v interface{}) zap.Field   { return zap.Any(k, v) }
