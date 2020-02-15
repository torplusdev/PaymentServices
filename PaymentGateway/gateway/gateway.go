package gateway

import (
	"github.com/stellar/go/protocols/horizon"
	"log"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
)

const seed = "SC37ORXN5DIXYWTJAB2I3N7245MLLRAJMRPK3EA5PPO6RTQLD5AD3XQU"

type Gateway struct {
	client *horizonclient.Client
	fullKeyPair keypair.Full
	account horizon.Account
}

func CreateGateway(useTestApi bool) *Gateway {
	gw := Gateway{}

	if useTestApi {
		gw.client = horizonclient.DefaultTestNetClient
	} else {
		gw.client = horizonclient.DefaultPublicNetClient
	}

	pair, err := keypair.ParseFull(seed)

	if err != nil {
		log.Fatal(err)
	}

	gw.fullKeyPair = *pair


	gwAccountDetail, errAccount := gw.client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID:pair.Address() })

	if errAccount != nil {
		log.Fatal(errAccount)
	}

	gw.account = gwAccountDetail

	return &gw
}

