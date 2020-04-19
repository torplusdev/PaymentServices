package common



type RouteAwarePaymentRequestValidator struct {

}

func CreateRouteAwarePaymentRequestValidator() *RouteAwarePaymentRequestValidator {
	v := RouteAwarePaymentRequestValidator{

	}

	return &v
}


func (validator *RouteAwarePaymentRequestValidator) Validate(pr PaymentRequest) (bool, string) {
	return true,""
}