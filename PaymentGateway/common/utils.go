package common

import (
	"context"
	"encoding/base64"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/plugin/httptrace"
	"io"
	"net/http"
)

type TraceContext struct {
	TraceID    string
	SpanID     string
	TraceFlags byte
}

func CreateTraceContext(ctx core.SpanContext) (TraceContext,error) {

	spanIdBytes,_ := ctx.SpanID.MarshalJSON()
	traceIdBytes,_ := ctx.TraceID.MarshalJSON()

	spanIdEncoded := base64.StdEncoding.EncodeToString(spanIdBytes)
	traceIdEncoded := base64.StdEncoding.EncodeToString(traceIdBytes)

	tc := TraceContext{
		TraceID:    traceIdEncoded,
		SpanID:     spanIdEncoded,
		TraceFlags: ctx.TraceFlags,
	}

	return tc,nil
}

func HttpGetWithContext(ctx context.Context, url string)  (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	ctx, req = httptrace.W3C(ctx, req)
	httptrace.Inject(ctx, req)

	return http.DefaultClient.Do(req)
}

func HttpPostWithContext(ctx context.Context, url string, body io.Reader)  (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)

	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type","application/json")

	ctx, req = httptrace.W3C(ctx, req)
	httptrace.Inject(ctx, req)

	return http.DefaultClient.Do(req)
}

func HttpPostWithoutContext(url string, body io.Reader)  (*http.Response, error) {
	req, _ := http.NewRequest("POST", url, body)
	req.Header.Add("Content-Type","application/json")

	return http.DefaultClient.Do(req)
}
