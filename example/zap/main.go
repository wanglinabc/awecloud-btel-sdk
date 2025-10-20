package main

import (
	"context"

	otlplogger "github.com/open-beagle/awecloud-btel-sdk/log"
	"github.com/open-beagle/awecloud-btel-sdk/log/otlpzap"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
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

	// logger := zap.New(otlpzap.NewCore())
	// defer logger.Sync()
	// logger.Info("hello world 1", zap.String("111", "222"))

	// tracer := otel.GetTracerProvider().Tracer("logrus")
	// ctx, span := tracer.Start(context.Background(), "logrus")
	// defer span.End()
	// logger.Info("hello world 2", zap.String("111", "222"), zap.Any("context", ctx))

	lg, _ := zap.NewDevelopment()
	logger := zap.New(otlpzap.NewCores(otlpzap.NewCore(), lg.Core()))
	defer logger.Sync()
	// logger.Info("hello world 1", zap.String("111", "222"))

	tracer := otel.GetTracerProvider().Tracer("logrus")
	ctx, span := tracer.Start(context.Background(), "logrus")
	defer span.End()
	logger.Info("hello world 2", zap.String("111", "222"), zap.Any("ctx", ctx))

	logger.With(zap.String("language", "rust")).Info("hello world 3", zap.String("111", "222"), zap.Any("ctx", ctx))
	logger.With(zap.String("owner", "go"), zap.String("language", "go")).Info("hello world 4", zap.String("111", "222"), zap.Any("ctx", ctx))
}
