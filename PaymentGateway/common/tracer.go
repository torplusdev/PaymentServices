package common

import (
	"paidpiper.com/payment-gateway/log"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	"paidpiper.com/payment-gateway/config"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func InitGlobalTracer(cfg *config.JaegerConfig) func() {
	if cfg == nil {
		return func() {}
	}
	// Create and install Jaeger export pipeline
	provider, flush, err := jaeger.NewExportPipeline(
		// http://192.168.162.128:14268/api/traces
		jaeger.WithCollectorEndpoint(cfg.Url),
		jaeger.WithProcess(jaeger.Process{
			ServiceName: cfg.ServiceName,
			Tags: []core.KeyValue{
				key.String("exporter", "jaeger"),
			},
		}),
		/// jaeger.RegisterAsGlobal() creates a lot of noise because of net/http traces, use it only if you really have to

		//jaeger.WithSDK(&sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),

		// NeverSample disables sampling
		jaeger.WithSDK(&sdktrace.Config{DefaultSampler: sdktrace.NeverSample()}),
	)

	if err != nil {
		log.Print("Could not connect to jaeger: " + err.Error())
	}

	traceProvider = provider
	return flush
}

var traceProvider trace.Provider

func CreateTracer(name string) trace.Tracer {

	if traceProvider != nil {
		return traceProvider.Tracer(name)
	}

	return trace.NoopTracer{}
}
