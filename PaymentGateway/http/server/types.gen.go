// Package server provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen DO NOT EDIT.
package server

const (
	SecurityRequirementScopes = "securityRequirement.Scopes"
)

// Balance defines model for Balance.
type Balance struct {
	Balance   *float64 `json:"balance,omitempty"`
	Timestamp *int64   `json:"timestamp,omitempty"`
}

// Error defines model for Error.
type Error struct {

	// Error code
	Code int32 `json:"code"`

	// Error message
	Message string `json:"message"`
}

// HistoryCollection defines model for HistoryCollection.
type HistoryCollection struct {
	Items []HistoryStatisticItem `json:"items"`
}

// HistoryStatisticItem defines model for HistoryStatisticItem.
type HistoryStatisticItem struct {

	// Date of unix
	Date int64 `json:"date"`

	// Value of bins
	Volume *int64 `json:"volume,omitempty"`
}

