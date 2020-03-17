package PaymentGateway

import (
	"github.com/stellar/go/keypair"
	"google.golang.org/grpc"
	"log"
	"net"
	"paidpiper.com/payment-gateway/proxy"
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

	key, _ := keypair.ParseFull("")

	pb.RegisterPPPaymentUtilityServicesServer(s, &us.UtilityServiceImpl{})
	pb.RegisterPPPaymentGatewayServer(s, &gw.GatewayServiceImpl{
		NodeManager: &proxy.NodeManager{},
		Seed: key,
	})

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
