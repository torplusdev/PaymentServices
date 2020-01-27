package testutils

import (
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/protocols/horizon"
	"log"
)

func CreateAndFundAccount(seed string) {

	client := horizonclient.DefaultTestNetClient

	pair, err := keypair.ParseFull(seed)

	if err != nil {
		log.Fatal(err)
	}

	_, errAccount := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID:pair.Address() })

	if errAccount != nil {
		txSuccess, errCreate := client.Fund(pair.Address())

		if errCreate != nil {
			log.Fatal(err)
		}

		log.Printf("Account " + seed + " created - trans#:",txSuccess.Hash)
	}
}

func GetAccount(address string)  (account horizon.Account, err error) {

	client := horizonclient.DefaultTestNetClient

	accountDetail, errAccount := client.AccountDetail(
		horizonclient.AccountRequest{
			AccountID:address})

	return accountDetail,errAccount
}
