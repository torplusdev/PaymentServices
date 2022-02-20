package common

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"go.opentelemetry.io/otel/plugin/httptrace"
	"paidpiper.com/payment-gateway/models"
)

func Error(status int, msg string) error {
	return &httpErrorMessage{
		Status: status,
		msg:    msg,
	}
}

type HttpErrorMessage interface {
	WriteHttpError(wr http.ResponseWriter) error
	Error() string
}
type httpErrorMessage struct {
	Status int
	msg    string
}

func (hem *httpErrorMessage) Error() string {
	return hem.msg
}

func (hem *httpErrorMessage) WriteHttpError(wr http.ResponseWriter) error {
	wr.WriteHeader(hem.Status)
	_, err := fmt.Fprintf(wr, hem.Error())
	return err

}

func HttpGetWithContext(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, err
	}

	ctx, req = httptrace.W3C(ctx, req)
	httptrace.Inject(ctx, req)

	return http.DefaultClient.Do(req)
}

func HttpPostWithContext(ctx context.Context, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)

	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	ctx, req = httptrace.W3C(ctx, req)
	httptrace.Inject(ctx, req)

	return http.DefaultClient.Do(req)
}

func HttpPostWithoutContext(url string, body io.Reader) (*http.Response, error) {
	req, _ := http.NewRequest("POST", url, body)
	req.Header.Add("Content-Type", "application/json")
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	return http.DefaultClient.Do(req)
}

func HttpPostWithoutResponseContext(url string, body io.Reader) error {
	req, _ := http.NewRequest("POST", url, body)
	req.Header.Add("Content-Type", "application/json")
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	_ = res.Body.Close() //??

	return nil
}

func HttpPaymentStatus(url string, body *models.PaymentStatusResponseModel) error {

	jsonValue, _ := json.Marshal(body)
	bytes.NewBuffer(jsonValue)
	return HttpPostWithoutResponseContext(url, bytes.NewBuffer(jsonValue))
}
