package integration_tests

import (
	"context"
	"fmt"
	"github.com/stellar/go/keypair"
	"log"
	"net/http"
	"paidpiper.com/payment-gateway/serviceNode"
	testutils "paidpiper.com/payment-gateway/tests"
	"time"
)

type TestSetup struct {
	servers []*http.Server
	torMock *testutils.TorMock
	torAddressPrefix string
}

func (setup *TestSetup) ConfigureTor(port int) {
	setup.torMock = testutils.CreateTorMock(port)
	setup.torAddressPrefix = fmt.Sprintf("http://localhost:%d",port)
}

func (setup *TestSetup) Shutdown() {

	if (setup.torMock != nil) {
		setup.torMock.Shutdown()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _,server := range setup.servers{
		server.Shutdown(ctx)
	}
}

func (setup *TestSetup) startNode(seed string, nodePort int) {
	srv,err := serviceNode.StartServiceNode(seed,nodePort,setup.torAddressPrefix)

	if err!=nil {
		log.Fatal("Coudn't start node")
	}
	setup.servers = append(setup.servers,srv)



}

func (setup *TestSetup) StartServiceNode(seed string, nodePort int) {
	setup.startNode(seed,nodePort)

	kp,_ := keypair.ParseFull(seed)

	if setup.torMock != nil {
		setup.torMock.RegisterNode(kp.Address(),nodePort)
	}
}

func (setup *TestSetup) StartTorNode(seed string, nodePort int) {
	setup.startNode(seed,nodePort)

	kp,_ := keypair.ParseFull(seed)

	if setup.torMock != nil {
		setup.torMock.RegisterTorNode(kp.Address(),nodePort)
	}
}

func (setup *TestSetup) StartUserNode(seed string, nodePort int) {
	setup.startNode(seed,nodePort)

	kp,_ := keypair.ParseFull(seed)

	if setup.torMock != nil {
		setup.torMock.RegisterNode(kp.Address(),nodePort)
	}
}

func CreateTestSetup() *TestSetup {

	setup := TestSetup{}

	return &setup
}