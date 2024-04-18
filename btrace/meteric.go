package btrace

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	metric1 "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"runtime"
	"strings"
	"time"
)

func (c *Tracer) initMeter(otlpEndpoint string) (success bool, err error) {
	var (
		metricExporter metric.Exporter
	)

	if otlpEndpoint == "stdout" {
		return

	} else if otlpEndpoint != "" {

		ifInsecure := true

		if strings.Contains(otlpEndpoint, "https") {
			ifInsecure = false
		}

		// 去除 http://
		otlpEndpoint = strings.TrimPrefix(otlpEndpoint, "http://")

		// 去除 https://
		otlpEndpoint = strings.TrimPrefix(otlpEndpoint, "https://")

		// 指定进行 http/protobuf 发送
		if c.OtelTracesExporter == "otlp" && c.OtelExporterOtlpProtocol == "http/protobuf" {
			splits := strings.Split(otlpEndpoint, "/")

			if len(splits) == 1 {
				if ifInsecure {

					metricExporter, err = otlpmetrichttp.New(c.metCtx,
						otlpmetrichttp.WithEndpoint(otlpEndpoint),
						otlpmetrichttp.WithInsecure())

				} else {
					metricExporter, err = otlpmetrichttp.New(c.metCtx, otlpmetrichttp.WithEndpoint(otlpEndpoint))
				}

			}

			if len(splits) > 1 {
				urlPath := "/v1/metrics"

				otlpEndpoint = splits[0]

				urlPath = strings.Join(splits[1:], "/") + "/v1/metrics"

				if ifInsecure {
					metricExporter, err = otlpmetrichttp.New(c.metCtx,
						otlpmetrichttp.WithEndpoint(otlpEndpoint),
						otlpmetrichttp.WithInsecure(),
						otlpmetrichttp.WithURLPath(urlPath))
				} else {
					metricExporter, err = otlpmetrichttp.New(c.metCtx,
						otlpmetrichttp.WithEndpoint(otlpEndpoint),
						otlpmetrichttp.WithURLPath(urlPath))
				}
			}

			if err != nil {
				return
			}

		} else {

			conn, err1 := grpc.DialContext(c.metCtx, otlpEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
			if err1 != nil {
				err = err1
				return
			}

			metricExporter, err = otlpmetricgrpc.New(c.metCtx, otlpmetricgrpc.WithGRPCConn(conn))
			if err != nil {
				return
			}
		}

		if metricExporter == nil {
			return
		}

		meterProvider := metric.NewMeterProvider(metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(60*time.Second))),
			metric.WithResource(c.Resource))

		otel.SetMeterProvider(meterProvider)

		c.stop = append(c.stop, func() {
			meterProvider.Shutdown(c.metCtx)
		})

		success = true

	}

	return

}

func captureMetric() error {

	meter := otel.GetMeterProvider().Meter("")

	_, err := meter.Int64ObservableUpDownCounter("go_memstats_sys_bytes", metric1.WithUnit("bytes"),
		metric1.WithInt64Callback(func(_ context.Context, obSrv metric1.Int64Observer) error {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			obSrv.Observe(int64(m.Sys))
			return nil
		}))

	return err

}
