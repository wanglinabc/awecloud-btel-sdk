package otlpzap

import (
	"context"
	"slices"

	_ "github.com/open-beagle/awecloud-btel-sdk/log"
	otlpResource "github.com/open-beagle/awecloud-btel-sdk/resource"
	"github.com/open-beagle/awecloud-btel-sdk/tool"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.uber.org/zap/zapcore"
)

type config struct {
	attributes  map[string]string
	serviceName string
	provider    log.LoggerProvider
	opts        []log.LoggerOption
	level       zapcore.Level
	enable      bool
}

func WithLoggerOption(options ...log.LoggerOption) optFunc {
	return func(cfg *config) {
		cfg.opts = options
	}
}

func WithLevel(level zapcore.Level) optFunc {
	return func(cfg *config) {
		cfg.level = level
	}
}

func WithEnable() optFunc {
	return func(cfg *config) {
		cfg.enable = true
	}
}

func WithAttributes(attributes map[string]string) optFunc {
	return func(cfg *config) {
		cfg.attributes = attributes
	}
}

func WithServiceName(serviceName string) optFunc {
	return func(cfg *config) {
		cfg.serviceName = serviceName
	}
}
func WithLoggerProvider(provider log.LoggerProvider) optFunc {
	return func(cfg *config) {
		cfg.provider = provider
	}
}

func newConfig(options []optFunc) *config {
	var c = &config{
		level:  zapcore.InfoLevel,
		enable: tool.GetLogExporterEnable(),
	}
	for _, opt := range options {
		opt(c)
	}
	if c.provider == nil {
		c.provider = global.GetLoggerProvider()
	}
	if c.serviceName == "" {
		c.serviceName = tool.GetServiceName()
	}
	c.mergeAttrs()
	return c
}
func (o *config) mergeAttrs() {
	if o.attributes == nil {
		o.attributes = map[string]string{}
	}
	serviceAttr := tool.GetAttributes()
	if len(serviceAttr) > 0 {
		for k, v := range serviceAttr {
			if _, ok := o.attributes[k]; !ok {
				o.attributes[k] = v
			}
		}
	}
}

type optFunc func(*config)

// Core is a [zapcore.Core] that sends logging records to OpenTelemetry.
type Core struct {
	cfg      *config
	logger   log.Logger
	resource *resource.Resource
	ctx      context.Context

	attr []log.KeyValue
}

var _ zapcore.Core = (*Core)(nil)

func NewCore(opts ...optFunc) zapcore.Core {
	cfg := newConfig(opts)

	// 创建资源
	attrs := []attribute.KeyValue{
		attribute.String("log.sdk", "betl-zap"),
	}
	// 添加自定义属性
	for k, v := range cfg.attributes {
		attrs = append(attrs, attribute.String(k, v))
	}
	res, _ := otlpResource.NewResourceWithAttributes(attrs...)
	// 创建 logger
	logger := cfg.provider.Logger("betl-zap", cfg.opts...)
	return &Core{
		cfg:      cfg,
		logger:   logger,
		resource: res,
		ctx:      context.Background(),
	}
}

// Enabled decides whether a given logging level is enabled when logging a message.
func (o *Core) Enabled(level zapcore.Level) bool {
	return o.cfg.enable && o.cfg.level.Enabled(level)
	// param := log.EnabledParameters{Severity: convertLevel(level)}
	// return o.logger.Enabled(context.Background(), param)
}

// With adds structured context to the Core.
func (o *Core) With(fields []zapcore.Field) zapcore.Core {
	cloned := o.clone()
	if len(fields) > 0 {
		ctx, attrbuf := convertField(fields)
		if ctx != nil {
			cloned.ctx = ctx
		}
		cloned.attr = append(cloned.attr, attrbuf...)
	}
	return cloned
}

func (o *Core) clone() *Core {
	return &Core{
		cfg:      o.cfg,
		resource: o.resource,
		logger:   o.logger,
		attr:     slices.Clone(o.attr),
		ctx:      o.ctx,
	}
}

// Sync flushes buffered logs (if any).
func (*Core) Sync() error {
	return nil
}

// Check determines whether the supplied Entry should be logged.
// If the entry should be logged, the Core adds itself to the CheckedEntry and returns the result.
func (o *Core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	return ce.AddCore(ent, o)
}

// Write method encodes zap fields to OTel logs and emits them.
func (o *Core) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	if !o.cfg.enable {
		return nil
	}
	r := log.Record{}
	r.SetTimestamp(ent.Time)
	r.SetBody(log.StringValue(ent.Message))
	r.SetSeverity(convertLevel(ent.Level))
	r.SetSeverityText(ent.Level.String())

	r.AddAttributes(o.attr...)
	if ent.Caller.Defined {
		r.AddAttributes(
			log.String(string(semconv.CodeFilePathKey), ent.Caller.File),
			log.Int(string(semconv.CodeLineNumberKey), ent.Caller.Line),
			log.String(string(semconv.CodeFunctionNameKey), ent.Caller.Function),
		)
	}
	if ent.Stack != "" {
		r.AddAttributes(log.String(string(semconv.CodeStacktraceKey), ent.Stack))
	}
	emitCtx := o.ctx
	if len(fields) > 0 {
		ctx, attrbuf := convertField(fields)
		if ctx != nil {
			emitCtx = ctx
		}
		r.AddAttributes(attrbuf...)
	}
	// 添加资源属性
	tool.AddResourceAttributes(o.resource, &r)

	o.logger.Emit(emitCtx, r)
	return nil
}

func convertField(fields []zapcore.Field) (context.Context, []log.KeyValue) {
	var ctx context.Context
	enc := newObjectEncoder(len(fields))
	for _, field := range fields {
		if ctxFld, ok := field.Interface.(context.Context); ok {
			ctx = ctxFld
			continue
		}
		field.AddTo(enc)
	}

	enc.calculate(enc.root)
	return ctx, enc.root.attrs
}

func convertLevel(level zapcore.Level) log.Severity {
	switch level {
	case zapcore.DebugLevel:
		return log.SeverityDebug
	case zapcore.InfoLevel:
		return log.SeverityInfo
	case zapcore.WarnLevel:
		return log.SeverityWarn
	case zapcore.ErrorLevel:
		return log.SeverityError
	case zapcore.DPanicLevel:
		return log.SeverityFatal1
	case zapcore.PanicLevel:
		return log.SeverityFatal2
	case zapcore.FatalLevel:
		return log.SeverityFatal3
	default:
		return log.SeverityUndefined
	}
}
