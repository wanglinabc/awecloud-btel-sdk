// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	otelresource "go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
)

const (
	// resourceAttrKey is the environment variable name OpenTelemetry Resource information will be read from.
	resourceAttrKey = "OTEL_RESOURCE_ATTRIBUTES"

	// svcNameKey is the environment variable name that Service Name information will be read from.
	svcNameKey = "BTEL_SERVICE_NAME"
)

var (
	// errMissingValue is returned when a resource value is missing.
	errMissingValue = fmt.Errorf("%w: missing value", otelresource.ErrPartialResource)
)

// FromEnv is a Detector that implements the Detector and collects
// resources from environment.  This Detector is included as a
// builtin.
type FromEnv struct{}

// compile time assertion that FromEnv implements Detector interface.
var _ otelresource.Detector = FromEnv{}

// Detect collects resources from environment.
func (FromEnv) Detect(context.Context) (*otelresource.Resource, error) {
	attrs := strings.TrimSpace(os.Getenv(resourceAttrKey))
	svcName := strings.TrimSpace(os.Getenv(svcNameKey))

	if attrs == "" && svcName == "" {
		return otelresource.Empty(), nil
	}

	var res *otelresource.Resource

	if svcName != "" {
		res = otelresource.NewSchemaless(semconv.ServiceNameKey.String(svcName))
	}

	r2, err := constructOTResources(attrs)

	// Ensure that the resource with the service name from OTEL_SERVICE_NAME
	// takes precedence, if it was defined.
	res, err2 := otelresource.Merge(r2, res)

	if err == nil {
		err = err2
	} else if err2 != nil {
		err = fmt.Errorf("detecting resources: %s", []string{err.Error(), err2.Error()})
	}

	return res, err
}

func constructOTResources(s string) (*otelresource.Resource, error) {
	if s == "" {
		return otelresource.Empty(), nil
	}
	pairs := strings.Split(s, ",")
	attrs := []attribute.KeyValue{}
	var invalid []string
	for _, p := range pairs {
		field := strings.SplitN(p, "=", 2)
		if len(field) != 2 {
			invalid = append(invalid, p)
			continue
		}
		k, v := strings.TrimSpace(field[0]), strings.TrimSpace(field[1])
		attrs = append(attrs, attribute.String(k, v))
	}
	var err error
	if len(invalid) > 0 {
		err = fmt.Errorf("%w: %v", errMissingValue, invalid)
	}
	return otelresource.NewSchemaless(attrs...), err
}
