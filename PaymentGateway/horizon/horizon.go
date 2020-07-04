package horizon

import (
	"github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/horizon"
	"paidpiper.com/payment-gateway/common"
)

type Horizon struct {
	Client *horizonclient.Client
}

func NewHorizon() *Horizon {
	return &Horizon{
		Client: NewHorizonClient(),
	}

}

func NewHorizonClient() *horizonclient.Client {
	return horizonclient.DefaultTestNetClient
}

func (horizon *Horizon) GetAccount(address string) (horizon.Account, error) {
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

func (horizon *Horizon) AddTransactionToken(tx *build.TransactionBuilder) error {
	return tx.Mutate(build.TestNetwork)
}
