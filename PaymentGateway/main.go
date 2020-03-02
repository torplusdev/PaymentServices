package PaymentGateway

import (
	"google.golang.org/grpc"
	"log"
	"net"
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

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
