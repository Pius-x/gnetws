package zaplog

import (
	"fmt"

	"go.uber.org/zap"
)

const (
	ErrMsg     = "err_msg"    // string 错误信息
	Stacktrace = "stacktrace" // string 错误堆栈
	TraceId    = "trace_id"   // string 日志追踪ID
	Latency    = "latency"    // int 耗时
)

func AppendErr(err error, fields ...zap.Field) []zap.Field {
	return append(fields, zap.String(ErrMsg, err.Error()), zap.String(Stacktrace, fmt.Sprintf("%+v", err)))
}
