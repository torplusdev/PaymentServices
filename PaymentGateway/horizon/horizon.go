package horizon

import (
	"github.com/stellar/go/clients/horizonclient"
	hProtocol "github.com/stellar/go/protocols/horizon"
	"paidpiper.com/payment-gateway/common"
)

type Horizon struct {
	Client *horizonclient.Client
}

func NewHorizon() *Horizon {
	client := &Horizon{
		Client: NewHorizonClient(),
	}

	//client.Client.HorizonURL = "https://stellar-test.bdnodes.net?auth=hFe2FLYQyk3rKKJ4oPsBG0ts--PFyPi5tEBBLffX1eU"

	return client
}

func NewHorizonClient() *horizonclient.Client {
	client := horizonclient.DefaultTestNetClient
	//client.HorizonURL = "https://stellar-test.bdnodes.net?auth=hFe2FLYQyk3rKKJ4oPsBG0ts--PFyPi5tEBBLffX1eU"

	return client
}

func (horizon *Horizon) GetAccount(address string) (hProtocol.Account, error) {
	return horizon.Client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID: address})
}

func (horizon *Horizon) GetBalance(address string) (string, error) {
	account, err := horizon.GetAccount(address)

	if err != nil {
		return "", err
	}

	balance := account.GetCreditBalance(common.PPTokenAssetName, common.PPTokenIssuerAddress)

	return balance, nil
}

//func (horizon *Horizon) AddTransactionToken(tx *build.TransactionBuilder) error {
//	return tx.Mutate(build.TestNetwork)
//}
