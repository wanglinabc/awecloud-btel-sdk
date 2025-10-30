package otlplogrus

import (
	"context"
	"fmt"

	otlpResource "github.com/open-beagle/awecloud-btel-sdk/resource"

	_ "github.com/open-beagle/awecloud-btel-sdk/log"
	"github.com/open-beagle/awecloud-btel-sdk/tool"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

var (
	levels = []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
		logrus.TraceLevel,
	}
)

// OTLPHook 将 logrus 日志转换为 OTLP 格式的 Hook
type OTLPHook struct {
	logger   log.Logger
	resource *resource.Resource
	config   *OTLPHookConfig
}
type OTLPHookOption func(cfg *OTLPHookConfig)

type OTLPHookConfig struct {
	serviceName string
	level       logrus.Level
	attributes  map[string]string
	provider    log.LoggerProvider
	enable      bool
	opts        []log.LoggerOption
}

func WithLevel(level logrus.Level) OTLPHookOption {
	return func(cfg *OTLPHookConfig) {
		cfg.level = level
	}
}

func WithLoggerOption(options ...log.LoggerOption) OTLPHookOption {
	return func(cfg *OTLPHookConfig) {
		cfg.opts = options
	}
}

func WithAttributes(attributes map[string]string) OTLPHookOption {
	return func(cfg *OTLPHookConfig) {
		cfg.attributes = attributes
	}
}

func WithServiceName(serviceName string) OTLPHookOption {
	return func(cfg *OTLPHookConfig) {
		cfg.serviceName = serviceName
	}
}
func WithLoggerProvider(provider log.LoggerProvider) OTLPHookOption {
	return func(cfg *OTLPHookConfig) {
		cfg.provider = provider
	}
}
func WithEnable() OTLPHookOption {
	return func(cfg *OTLPHookConfig) {
		cfg.enable = true
	}
}

func (o *OTLPHookConfig) mergeAttrs() {
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

// NewHook 创建新的 OTLP Hook
func NewHook(opts ...OTLPHookOption) logrus.Hook {
	cfg := &OTLPHookConfig{
		level:  logrus.InfoLevel,
		enable: tool.GetLogExporterEnable(),
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.serviceName == "" {
		cfg.serviceName = tool.GetServiceName()
	}
	if cfg.provider == nil {
		cfg.provider = global.GetLoggerProvider()
	}
	cfg.mergeAttrs()

	// 创建资源
	attrs := []attribute.KeyValue{
		attribute.String("log.sdk", "betl-logrus"),
	}
	// 添加自定义属性
	for k, v := range cfg.attributes {
		attrs = append(attrs, attribute.String(k, v))
	}
	res, _ := otlpResource.NewResourceWithAttributes(attrs...)
	// 创建 logger
	logger := cfg.provider.Logger("betl-logrus", cfg.opts...)
	return &OTLPHook{
		logger:   logger,
		resource: res,
		config:   cfg,
	}
}

// Levels 指定处理的日志级别
func (h *OTLPHook) Levels() []logrus.Level {
	if h.config.level >= logrus.TraceLevel {
		return levels
	}
	return levels[0 : h.config.level+1]
}

func (h *OTLPHook) extratchEntryField(ctx context.Context, entry *logrus.Entry) bool {
	flag := false
	if v, ok := entry.Data["traceId"]; !ok || v == "" {
		traceID, spanID := tool.GetTraceAndSpanID(ctx)
		if traceID != "" {
			entry.Data["traceId"] = traceID
			entry.Data["spanId"] = spanID
			flag = true
		}
	}
	if v, ok := entry.Data["serviceName"]; !ok || v == "" {
		if h.config.serviceName != "" {
			entry.Data["serviceName"] = h.config.serviceName
		}
	}
	return flag
}

// Fire 处理日志条目
func (h *OTLPHook) Fire(entry *logrus.Entry) error {
	// 转换 logrus entry 为 OTLP log record
	ctx := context.Background()
	if entry.Context != nil {
		ctx = entry.Context
	}

	h.extratchEntryField(ctx, entry)
	if !h.config.enable {
		return nil
	}
	record := h.convertToOTLPRecord(entry)
	h.logger.Emit(ctx, record)
	return nil
}

// convertToOTLPRecord 将 logrus Entry 转换为 OTLP Log Record
func (h *OTLPHook) convertToOTLPRecord(entry *logrus.Entry) log.Record {
	var record log.Record

	// 设置时间戳
	record.SetTimestamp(entry.Time)

	// 设置日志级别
	severity := h.convertSeverity(entry.Level)
	record.SetSeverity(severity)
	record.SetSeverityText(entry.Level.String())

	// 设置日志消息体
	if entry.Message != "" {
		record.SetBody(log.StringValue(entry.Message))
	}

	if entry.Caller != nil {
		record.AddAttributes(
			log.String(string(semconv.CodeFilePathKey), entry.Caller.File),
			log.Int(string(semconv.CodeLineNumberKey), entry.Caller.Line),
			log.String(string(semconv.CodeFunctionNameKey), entry.Caller.Function),
		)
	}

	// 添加属性
	attrs := h.convertFieldsToAttributes(entry.Data)
	for _, attr := range attrs {
		record.AddAttributes(attr)
	}
	// 添加资源属性
	tool.AddResourceAttributes(h.resource, &record)

	return record
}

// convertSeverity 转换 logrus 级别为 OTLP 级别
func (h *OTLPHook) convertSeverity(level logrus.Level) log.Severity {
	switch level {
	case logrus.TraceLevel:
		return log.SeverityTrace
	case logrus.DebugLevel:
		return log.SeverityDebug
	case logrus.InfoLevel:
		return log.SeverityInfo
	case logrus.WarnLevel:
		return log.SeverityWarn
	case logrus.ErrorLevel:
		return log.SeverityError
	case logrus.FatalLevel:
		return log.SeverityFatal
	case logrus.PanicLevel:
		return log.SeverityFatal
	default:
		return log.SeverityUndefined
	}
}

// convertFieldsToAttributes 转换 logrus fields 为 OTLP attributes
func (h *OTLPHook) convertFieldsToAttributes(fields logrus.Fields) []log.KeyValue {
	var attrs []log.KeyValue

	for key, value := range fields {
		// fmt.Printf("convertFieldsToAttributes %+v\n", key)
		// 跳过一些标准字段
		if tool.InArray([]string{"time", "msg", "level", "traceId", "spanId", "serviceName"}, key) {
			continue
		}
		attr := h.convertValueToAttribute(key, value)
		if attr.Key != "" {
			attrs = append(attrs, attr)
		}
	}

	return attrs
}

// convertValueToAttribute 转换值为 OTLP attribute
func (h *OTLPHook) convertValueToAttribute(key string, value interface{}) log.KeyValue {
	switch v := value.(type) {
	case string:
		return log.String(key, v)
	case int:
		return log.Int(key, v)
	case int64:
		return log.Int64(key, v)
	case float64:
		return log.Float64(key, v)
	case bool:
		return log.Bool(key, v)
	case error:
		return log.String(key, v.Error())
	case []byte:
		return log.String(key, string(v))
	default:
		// 对于其他类型，转换为字符串
		return log.String(key, fmt.Sprintf("%v", v))
	}
}

// addResourceAttributes 添加资源属性
