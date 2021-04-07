package controllers

import (
	"context"
	"encoding/base64"
	"net/http"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/correlation"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/plugin/httptrace"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/models"
)

func spanFromContext(rootContext context.Context, traceContext *models.TraceContext, spanName string) (context.Context, trace.Span) {

	tracer := common.CreateTracer("paidpiper/controller")

	var traceId [16]byte
	var spanId [8]byte

	ba, _ := base64.StdEncoding.DecodeString(traceContext.TraceID)
	copy(traceId[:], ba)

	ba, _ = base64.StdEncoding.DecodeString(traceContext.SpanID)
	copy(spanId[:], ba)

	var span trace.Span
	var ctx context.Context

	spanContext := core.SpanContext{
		TraceID:    traceId,
		SpanID:     spanId,
		TraceFlags: traceContext.TraceFlags,
	}

	ctx, span = tracer.Start(
		trace.ContextWithRemoteSpanContext(rootContext, spanContext),
		spanName,
	)

	return ctx, span
}

func spanFromRequest(r *http.Request, spanName string) (context.Context, trace.Span) {

	tracer := common.CreateTracer("paidpiper/controller")
	attrs, entries, spanCtx := httptrace.Extract(r.Context(), r)

	r = r.WithContext(correlation.ContextWithMap(r.Context(), correlation.NewMap(correlation.MapUpdate{
		MultiKV: entries,
	})))

	ctx, span := tracer.Start(
		trace.ContextWithRemoteSpanContext(r.Context(), spanCtx),
		spanName,
		trace.WithAttributes(attrs...),
	)

	return ctx, span
}
