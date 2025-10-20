package otlpzap

import (
	"context"
	"slices"

	"github.com/open-beagle/awecloud-btel-sdk/tool"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type multiCore []zapcore.Core

var (
	_ zapcore.Core = multiCore(nil)
)

// NewCores creates a Core that duplicates log entries into two or more
// underlying Cores.
//
// Calling it with a single Core returns the input unchanged, and calling
// it with no input returns a no-op Core.
func NewCores(cores ...zapcore.Core) zapcore.Core {
	switch len(cores) {
	case 0:
		return NewCore()
	case 1:
		return cores[0]
	default:
		return multiCore(cores)
	}
}

func (mc multiCore) With(fields []zap.Field) zapcore.Core {
	clone := make(multiCore, len(mc))
	for i := range mc {
		clone[i] = mc[i].With(fields)
	}
	return clone
}

func (mc multiCore) Enabled(lvl zapcore.Level) bool {
	for i := range mc {
		if mc[i].Enabled(lvl) {
			return true
		}
	}
	return false
}

func (mc multiCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	return ce.AddCore(ent, mc)
}

func (mc multiCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	//判断是否有context字段
	var ctx context.Context
	var index = -1
	for i, field := range fields {
		if field.Key == "ctx" || field.Key == "context" {
			if ctxFld, ok := field.Interface.(context.Context); ok {
				ctx = ctxFld
				index = i
				break
			}
		}

	}
	newFields := slices.Clone(fields)
	if ctx != nil {
		traceID, spanID := tool.GetTraceAndSpanID(ctx)
		if traceID != "" {
			newFields = append(newFields, zap.String("traceId", traceID), zap.String("spanId", spanID), zap.String("serviceName", tool.GetServiceName()))
		}
	}
	var err error
	for i := range mc {
		if mc[i].Enabled(ent.Level) {
			if _, ok := mc[i].(*Core); !ok {
				err = multierr.Append(err, mc[i].Write(ent, tool.RemoveArrayIndexOrdered(newFields, index)))
			} else {
				err = multierr.Append(err, mc[i].Write(ent, fields))
			}
		}
	}
	return err
}

func (mc multiCore) Sync() error {
	var err error
	for i := range mc {
		err = multierr.Append(err, mc[i].Sync())
	}
	return err
}
