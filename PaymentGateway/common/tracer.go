package common

import "go.opentelemetry.io/otel/api/trace"

var traceProvider trace.Provider

func InitializeTracer(provider trace.Provider) {
	traceProvider = provider
}

func CreateTracer(name string) trace.Tracer {

	if  traceProvider != nil {
		return traceProvider.Tracer(name)
	}

	return trace.NoopTracer{}
}