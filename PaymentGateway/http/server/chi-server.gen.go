// Package server provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen DO NOT EDIT.
package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/deepmap/oapi-codegen/pkg/runtime"
	"github.com/go-chi/chi/v5"
)

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// Get balance
	// (GET /api/v1/book/balance)
	GetBalance(w http.ResponseWriter, r *http.Request)
	// Get history
	// (GET /api/v1/book/history/{commodity}/{hours}/{bins})
	GetHistory(w http.ResponseWriter, r *http.Request, commodity string, hours int32, bins int32)
}

// ServerInterfaceWrapper converts contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler            ServerInterface
	HandlerMiddlewares []MiddlewareFunc
}

type MiddlewareFunc func(http.HandlerFunc) http.HandlerFunc

// GetBalance operation middleware
func (siw *ServerInterfaceWrapper) GetBalance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ctx = context.WithValue(ctx, SecurityRequirementScopes, []string{""})

	var handler = func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetBalance(w, r)
	}

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler(w, r.WithContext(ctx))
}

// GetHistory operation middleware
func (siw *ServerInterfaceWrapper) GetHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var err error

	// ------------- Path parameter "commodity" -------------
	var commodity string

	err = runtime.BindStyledParameter("simple", false, "commodity", chi.URLParam(r, "commodity"), &commodity)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid format for parameter commodity: %s", err), http.StatusBadRequest)
		return
	}

	// ------------- Path parameter "hours" -------------
	var hours int32

	err = runtime.BindStyledParameter("simple", false, "hours", chi.URLParam(r, "hours"), &hours)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid format for parameter hours: %s", err), http.StatusBadRequest)
		return
	}

	// ------------- Path parameter "bins" -------------
	var bins int32

	err = runtime.BindStyledParameter("simple", false, "bins", chi.URLParam(r, "bins"), &bins)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid format for parameter bins: %s", err), http.StatusBadRequest)
		return
	}

	ctx = context.WithValue(ctx, SecurityRequirementScopes, []string{""})

	var handler = func(w http.ResponseWriter, r *http.Request) {
		siw.Handler.GetHistory(w, r, commodity, hours, bins)
	}

	for _, middleware := range siw.HandlerMiddlewares {
		handler = middleware(handler)
	}

	handler(w, r.WithContext(ctx))
}

// Handler creates http.Handler with routing matching OpenAPI spec.
func Handler(si ServerInterface) http.Handler {
	return HandlerWithOptions(si, ChiServerOptions{})
}

type ChiServerOptions struct {
	BaseURL     string
	BaseRouter  chi.Router
	Middlewares []MiddlewareFunc
}

// HandlerFromMux creates http.Handler with routing matching OpenAPI spec based on the provided mux.
func HandlerFromMux(si ServerInterface, r chi.Router) http.Handler {
	return HandlerWithOptions(si, ChiServerOptions{
		BaseRouter: r,
	})
}

func HandlerFromMuxWithBaseURL(si ServerInterface, r chi.Router, baseURL string) http.Handler {
	return HandlerWithOptions(si, ChiServerOptions{
		BaseURL:    baseURL,
		BaseRouter: r,
	})
}

// HandlerWithOptions creates http.Handler with additional options
func HandlerWithOptions(si ServerInterface, options ChiServerOptions) http.Handler {
	r := options.BaseRouter

	if r == nil {
		r = chi.NewRouter()
	}
	wrapper := ServerInterfaceWrapper{
		Handler:            si,
		HandlerMiddlewares: options.Middlewares,
	}

	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/v1/book/balance", wrapper.GetBalance)
	})
	r.Group(func(r chi.Router) {
		r.Get(options.BaseURL+"/api/v1/book/history/{commodity}/{hours}/{bins}", wrapper.GetHistory)
	})

	return r
}
