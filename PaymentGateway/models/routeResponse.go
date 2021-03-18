package models

type RouteResponse struct {
	Route []RoutingNode

	CallbackUrl string // Payment command url

	StatusCallbackUrl string // Status callback command url
}
