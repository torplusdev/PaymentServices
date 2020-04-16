package integration_tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/stellar/go/keypair"
	"io/ioutil"
	"log"
	"net/http"
	"paidpiper.com/payment-gateway/common"
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

func (setup *TestSetup) CreatePaymentInfo(seed string, amount int) (common.PaymentRequest,error) {

	kp,_ := keypair.ParseFull(seed)

	port := setup.torMock.GetNodePort(kp.Address())

	resp,err := http.Get(fmt.Sprintf("http://localhost:%d/api/utility/createPaymentInfo/%d", port, amount))

	if err != nil || resp.StatusCode != http.StatusOK {
		return common.PaymentRequest{}, err
	}

	dec := json.NewDecoder(resp.Body)

	var pr common.PaymentRequest
	dec.Decode(&pr)

	return pr,nil
}

func (setup *TestSetup) FlushTransactions() error {

	for _,v := range setup.torMock.GetNodes() {
		resp,err := http.Get(fmt.Sprintf("http://localhost:%d/api/utility/flushTransactions", v))

		if err != nil || resp.StatusCode != http.StatusOK {
			return err
		}
	}

	return nil
}

type ProcessPaymentRequest struct {
	RouteAddresses       []string
	CallbackUrl			 string
	PaymentRequest		 string
}

func (setup *TestSetup) ProcessPayment(seed string,paymentRequest common.PaymentRequest) (string, error) {

	kp,_ := keypair.ParseFull(seed)

	port := setup.torMock.GetNodePort(kp.Address())

	prBytes,err := json.Marshal(paymentRequest)

	ppr := ProcessPaymentRequest{
		RouteAddresses: []string{},
		CallbackUrl: "",
		PaymentRequest:  string(prBytes),
	}

	pprBytes,err := json.Marshal(ppr)

	resp,err := http.Post(fmt.Sprintf("http://localhost:%d/api/gateway/processPayment", port),"application/json",bytes.NewReader(pprBytes))

	if err != nil || resp.StatusCode != http.StatusOK {
		return "error", err
	}

	respByte, err := ioutil.ReadAll(resp.Body)

	if err != nil  {
		return "error", err
	}

	result := string(respByte)

	return result, nil
}


func CreateTestSetup() *TestSetup {

	setup := TestSetup{}

	return &setup
}