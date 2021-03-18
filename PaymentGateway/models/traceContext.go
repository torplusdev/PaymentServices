package models

import (
	"encoding/base64"

	"go.opentelemetry.io/otel/api/core"
)

type TraceContext struct {
	TraceID    string
	SpanID     string
	TraceFlags byte
}

func NewTraceContext(ctx core.SpanContext) (*TraceContext, error) {

	spanIdBytes, err := ctx.SpanID.MarshalJSON()
	if err != nil {
		return nil, err
	}
	traceIdBytes, err := ctx.TraceID.MarshalJSON()
	if err != nil {
		return nil, err
	}
	spanIdEncoded := base64.StdEncoding.EncodeToString(spanIdBytes)
	traceIdEncoded := base64.StdEncoding.EncodeToString(traceIdBytes)

	tc := &TraceContext{
		TraceID:    traceIdEncoded,
		SpanID:     spanIdEncoded,
		TraceFlags: ctx.TraceFlags,
	}

	return tc, nil
}
