// Package otlplogger provides OpenTelemetry logging functionality.
package log

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/open-beagle/awecloud-btel-sdk/tool"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	otel_log "go.opentelemetry.io/otel/sdk/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func init() {
	_ = NewLogProvider()
}

type ProviderConfig struct {
	exporter otel_log.Exporter
	debug    bool
}
type ProviderOption func(cfg *ProviderConfig)

func WithExporter(exporter otel_log.Exporter) ProviderOption {
	return func(cfg *ProviderConfig) {
		cfg.exporter = exporter
	}
}
func WithDebug() ProviderOption {
	return func(cfg *ProviderConfig) {
		cfg.debug = true
	}
}
func NewLogProvider(opts ...ProviderOption) log.LoggerProvider {
	cfg := &ProviderConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.exporter == nil {
		cfg.exporter = newDefaultExporter(cfg)
	}
	var (
		processor otel_log.Processor
	)
	processor = otel_log.NewBatchProcessor(cfg.exporter)
	if cfg.debug {
		processor = otel_log.NewSimpleProcessor(cfg.exporter)
	}

	logProvider := otel_log.NewLoggerProvider(
		otel_log.WithProcessor(
			processor,
		),
	)
	global.SetLoggerProvider(logProvider)
	return logProvider
}

func newDefaultExporter(cfg *ProviderConfig) otel_log.Exporter {
	exporter, err := initOtlpExport()
	if err == nil {
		return exporter
	}
	fmt.Println("init initOtlpExport  Err:", err.Error())
	if cfg.debug {
		return newDebugExporter()
	}
	exporter, err = stdoutlog.New(stdoutlog.WithPrettyPrint())
	if err == nil {
		return exporter
	}
	fmt.Println("init stdoutlog  Err:", err.Error())
	return newDebugExporter()
}
func initOtlpExport() (otel_log.Exporter, error) {
	if otlpEndpoint := tool.GetCollectorEndpoint(); otlpEndpoint != "" && otlpEndpoint != "stdout" {
		ifInsecure := true
		if strings.Contains(otlpEndpoint, "https") {
			ifInsecure = false
		}
		// 去除 http://
		otlpEndpoint = strings.TrimPrefix(otlpEndpoint, "http://")
		// 去除 https://
		otlpEndpoint = strings.TrimPrefix(otlpEndpoint, "https://")

		switch strings.ToLower(tool.GetCollectorExporterProtocol()) {
		case "grpc":
			if ifInsecure {
				return otlploggrpc.New(context.Background(), otlploggrpc.WithEndpoint(otlpEndpoint), otlploggrpc.WithInsecure(), otlploggrpc.WithDialOption(grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials())))
			} else {
				return otlploggrpc.New(context.Background(), otlploggrpc.WithEndpoint(otlpEndpoint), otlploggrpc.WithDialOption(grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials())))
			}
		default:
			//http/protobuf
			if ifInsecure {
				return otlploghttp.New(context.Background(), otlploghttp.WithEndpoint(otlpEndpoint), otlploghttp.WithInsecure())
			} else {
				return otlploghttp.New(context.Background(), otlploghttp.WithEndpoint(otlpEndpoint))
			}
		}
	}
	return nil, errors.New("otel endpoint not init yet")
}

func newDebugExporter() otel_log.Exporter {
	return &debugExporter{}
}

type debugExporter struct {
}

func (e *debugExporter) Export(ctx context.Context, records []otel_log.Record) error {
	for _, record := range records {
		fmt.Printf("traceId: %s\n", record.TraceID().String())
		fmt.Printf("spanId: %s\n", record.SpanID().String())
		fmt.Printf("body: %s\n", record.Body().AsString())
		record.WalkAttributes(func(attr log.KeyValue) bool {
			fmt.Printf("attr: %+v\n", attr)
			return true
		})
	}
	return nil
}

func (e *debugExporter) Shutdown(ctx context.Context) error {
	return nil
}

func (e *debugExporter) ForceFlush(ctx context.Context) error {
	return nil
}
