package resource

import (
	"context"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func NewResource(attrs ...attribute.KeyValue) (*resource.Resource, error) {
	resources, err := resource.New(context.Background(),
		// WithFromEnv(), // pull attributes from OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables
		WithOtherProcess(),
		resource.WithProcess(), // This option configures a set of Detectors that discover process information
		resource.WithAttributes(attrs...),
	)
	return resources, err
}

func GetDefaultResource(serviceName string) *resource.Resource {
	hostname, _ := os.Hostname()
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(serviceName),
		semconv.HostNameKey.String(hostname),
		semconv.ProcessPIDKey.Int(os.Getpid()),
		semconv.ProcessCommandKey.String(os.Args[0]),
	)
}
