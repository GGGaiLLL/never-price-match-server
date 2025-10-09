package logger

import "go.uber.org/zap"

func Field(k string, v interface{}) zap.Field { return zap.Any(k, v) }
func Err(err error) zap.Field                 { return zap.Error(err) }
func Str(k, v string) zap.Field               { return zap.String(k, v) }
func Int(k string, v int) zap.Field           { return zap.Int(k, v) }
func Dur(k string, v interface{}) zap.Field   { return zap.Any(k, v) }
