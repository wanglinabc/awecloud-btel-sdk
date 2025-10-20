package tool

import (
	"context"
	"os"
	"strings"

	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/trace"
)

func GetServiceName() string {
	return GetEnvString("BTEL_SERVICE_NAME", os.Args[0])
}

func GetEnvString(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func GetCollectorEndpoint() string {
	return GetEnvString("BTEL_EXPORTER_OTLP_ENDPOINT", "")
}
func GetCollectorExporterProtocol() string {
	return GetEnvString("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
}
func GetCollectorID() string {
	return GetEnvString("COLLECTOR_ID", "")
}

// func GetDeployment() string {
// 	return GetEnvString("EXPORTER_ENDPOINT", "http://localhost:4317")
// }
// func GetClusterId() string {
// 	return GetEnvString("EXPORTER_ENDPOINT", "http://localhost:4317")
// }
// func GetNameSpace() string {
// 	return GetEnvString("EXPORTER_ENDPOINT", "http://localhost:4317")
// }

func GetCollectorInfo() (string, string, string) {
	var deployment, namespace, cluster string
	if value := GetEnvString("OTEL_RESOURCE_ATTRIBUTES", ""); value != "" {
		attributes := strings.Split(value, ",")
		for _, attribute := range attributes {
			attrs := strings.Split(attribute, "=")
			if len(attrs) != 2 {
				continue
			}
			if attrs[0] == "deployment" {
				deployment = attrs[1]
			}

			if attrs[0] == "namespace" {
				namespace = attrs[1]
			}

			if attrs[0] == "cluster" {
				cluster = attrs[1]
			}
		}
	}
	return deployment, namespace, cluster
}

func GetAttributes() map[string]string {
	res := make(map[string]string, 10)
	if v := GetCollectorID(); v != "" {
		res["service.collectorId"] = v
	}
	deployment, namespace, cluster := GetCollectorInfo()
	if deployment != "" {
		res["service.deployment"] = deployment
	}
	if namespace != "" {
		res["service.namespace"] = namespace
	}
	if cluster != "" {
		res["service.cluster"] = cluster
	}
	return res
}

func GetTraceAndSpanID(ctx context.Context) (string, string) {
	sc := trace.SpanContextFromContext(ctx)
	if sc.IsValid() {
		return sc.TraceID().String(), sc.SpanID().String()
	}
	return "", ""
}

func InArray[T comparable](arr []T, elem T) bool {
	for _, v := range arr {
		if v == elem {
			return true
		}
	}
	return false
}

func RemoveArrayIndexOrdered[S ~[]T, T any](s S, index int) S {
	if index < 0 || index >= len(s) {
		return s
	}
	if index == 0 {
		return s[1:]
	}
	if index == len(s)-1 {
		return s[0 : index-1]
	}
	return append(s[:index], s[index+1:]...)
}

func AddResourceAttributes(res *resource.Resource, record *log.Record) {
	if res != nil && record != nil {
		for _, attr := range res.Attributes() {
			record.AddAttributes(log.KeyValue{
				Key:   string(attr.Key),
				Value: log.ValueFromAttribute(attr.Value),
			})
		}
	}
}
