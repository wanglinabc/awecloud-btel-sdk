package otlplogrus

import (
	"sync"

	"github.com/sirupsen/logrus"
)

const (
	traceKey       = "traceId"
	spanKey        = "spanId"
	serviceNameKey = "serviceName"
)

type OtelFormat struct {
	Formater   logrus.Formatter
	TraceKey   string
	SpanKey    string
	ServiceKey string
	once       sync.Once
}

// Format implements logrus.Formatter.
func (o *OtelFormat) Format(entry *logrus.Entry) ([]byte, error) {
	o.once.Do(o.init)
	if v, ok := entry.Data[traceKey]; ok && v != "" && traceKey != o.TraceKey {
		entry.Data[o.TraceKey] = v
		delete(entry.Data, traceKey)
	}
	if v, ok := entry.Data[spanKey]; ok && v != "" && spanKey != o.SpanKey {
		entry.Data[o.SpanKey] = v
		delete(entry.Data, spanKey)
	}
	if v, ok := entry.Data[serviceNameKey]; ok && v != "" && serviceNameKey != o.ServiceKey {
		entry.Data[o.ServiceKey] = v
		delete(entry.Data, serviceNameKey)
	}
	return o.Formater.Format(entry)
}

func (o *OtelFormat) init() {
	if o.TraceKey == "" {
		o.TraceKey = traceKey
	}
	if o.SpanKey == "" {
		o.SpanKey = spanKey
	}
	if o.ServiceKey == "" {
		o.ServiceKey = serviceNameKey
	}
}

var (
	_ logrus.Formatter = (*OtelFormat)(nil)
)
