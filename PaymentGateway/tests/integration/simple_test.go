package integration_tests

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"os"
	"paidpiper.com/payment-gateway/common"
	testutils "paidpiper.com/payment-gateway/tests"
	"testing"
	"time"
)

func setup() {

	// Addresses reused from other tests
	testutils.CreateAndFundAccount(testutils.User1Seed)
	testutils.CreateAndFundAccount(testutils.Service1Seed)

	// Addresses specific to this test suite
	testutils.CreateAndFundAccount(testutils.Node1Seed)
	testutils.CreateAndFundAccount(testutils.Node2Seed)
	testutils.CreateAndFundAccount(testutils.Node3Seed)
	testutils.CreateAndFundAccount(testutils.Node4Seed)
}

func shutdown() {

}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}


func TestSimple(t *testing.T) {
	assert := assert.New(t)

	torPort := 57842


	testSetup := CreateTestSetup()
	defer testSetup.Shutdown()

	testSetup.ConfigureTor(torPort)

	testSetup.StartUserNode(testutils.User1Seed,28080)
	testSetup.StartTorNode(testutils.Node1Seed,28081)
	testSetup.StartTorNode(testutils.Node2Seed,28082)
	testSetup.StartTorNode(testutils.Node3Seed,28083)
	testSetup.StartServiceNode(testutils.Service1Seed,28084)

	// Wait for everything to start up
	time.Sleep(2 * time.Second)

	resp,err := http.Get("http://localhost:28084/api/utility/createPaymentInfo/10")

	assert.NoError(err)
	assert.True(resp.StatusCode == http.StatusOK)
	dec := json.NewDecoder(resp.Body)

	var pr common.PaymentRequest
	assert.NoError(dec.Decode(&pr))

	type ProcessPaymentRequest struct {
		RouteAddresses       []string
		CallbackUrl			 string
		PaymentRequest		 string
	}

	prBytes,err := json.Marshal(pr)
	assert.NoError(err)

	ppr := ProcessPaymentRequest{
		RouteAddresses: []string{},
		CallbackUrl: "",
		PaymentRequest:  string(prBytes),
	}

	pprBytes,err := json.Marshal(ppr)
	assert.NoError(err)

	resp,err = http.Post("http://localhost:28080/api/gateway/processPayment","application/json",bytes.NewReader(pprBytes))
	assert.NoError(err)
	assert.True(resp.StatusCode == http.StatusOK)

	respByte, err := ioutil.ReadAll(resp.Body)
	assert.NoError(err)
	result := string(respByte)
	assert.True(resp.StatusCode == http.StatusOK)

	_ = result
}