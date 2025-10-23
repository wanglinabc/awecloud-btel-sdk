package resource

import (
	"context"
	"os"

	"github.com/open-beagle/awecloud-btel-sdk/tool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func NewResourceWithAttributes(attrs ...attribute.KeyValue) (*resource.Resource, error) {
	resources, err := resource.New(context.Background(),
		resource.WithAttributes(attrs...),
	)
	return resources, err
}

func NewDefaultResource() (*resource.Resource, error) {
	hostname, _ := os.Hostname()
	resources, err := resource.New(context.Background(),
		WithOtherProcess(),
		resource.WithProcess(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(tool.GetServiceName()),
			semconv.HostNameKey.String(hostname),
			semconv.ProcessPIDKey.Int(os.Getpid()),
			semconv.ProcessCommandKey.String(os.Args[0])),
	)
	return resources, err
}
