package main

import (
	"context"
	"fmt"

	otlplogger "github.com/open-beagle/awecloud-btel-sdk/log"
	"github.com/open-beagle/awecloud-btel-sdk/log/otlplogrus"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func initTrace() {
	otel.SetTracerProvider(
		sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
		),
	)

}
func initProvider() {
	_ = otlplogger.NewLogProvider(otlplogger.WithDebug())

}

func main() {
	initTrace()

	initProvider()

	logrus.SetReportCaller(true)

	logrus.AddHook(otlplogrus.NewHook(otlplogrus.WithEnable()))
	logrus.WithContext(context.Background()).Info("我来修改导出器1")

	// ctx := context.Background()
	tracer := otel.GetTracerProvider().Tracer("logrus")
	ctx, span := tracer.Start(context.Background(), "logrus")

	defer span.End()

	// spanContext := trace.SpanContextFromContext(ctx)
	// fmt.Println("traceId=", spanContext.TraceID().String())
	// fmt.Println("spanId=", spanContext.SpanID().String())

	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.WithContext(ctx).WithField("hello", "world").Info("打印日志")
	logrus.SetFormatter(&otlplogrus.OtelFormat{
		Formater: &logrus.JSONFormatter{},
		TraceKey: "trace",
	})
	logrus.WithContext(ctx).Info("我来修改导出器2")
}

type stdOutExporter struct{}

func (e *stdOutExporter) Export(ctx context.Context, records []sdklog.Record) error {
	fmt.Println("=== OTLP Export called ===")
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

func (e *stdOutExporter) Shutdown(ctx context.Context) error {
	return nil
}

func (e *stdOutExporter) ForceFlush(ctx context.Context) error {
	return nil
}

type stdHook struct {
}

func (h *stdHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.InfoLevel,
	}
}

func (h *stdHook) Fire(entry *logrus.Entry) error {
	fmt.Printf("%+v\n", "stdHook")
	return nil
}
