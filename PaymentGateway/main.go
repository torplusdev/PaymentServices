package PaymentGateway

import (
	"google.golang.org/grpc"
	"log"
	"net"
	gw "paidpiper.com/payment-gateway/gatewayService"
	pb "paidpiper.com/payment-gateway/ppsidechannel"
	us "paidpiper.com/payment-gateway/utilityService"
)

func main() {
	lis, err := net.Listen("tcp", ":28080")

	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()

	pb.RegisterPPPaymentUtilityServicesServer(s, &us.UtilityServiceImpl{})
	pb.RegisterPPPaymentGatewayServer(s, &gw.GatewayServiceImpl{})

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
