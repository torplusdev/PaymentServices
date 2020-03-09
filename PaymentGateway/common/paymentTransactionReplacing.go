package common

import (
	"errors"
	"log"
)

type PaymentTransactionReplacing struct {
	pendingTransaction PaymentTransaction
	referenceTransaction PaymentTransaction
}

func CreateReferenceTransaction (pt PaymentTransaction, ref PaymentTransaction) (PaymentTransactionReplacing, error) {
	if ref.XDR != "" {
		if pt.PaymentDestinationAddress != ref.PaymentDestinationAddress {
			log.Print("Error creating accumulating transactions, two transactions have different destination addresses")
			return PaymentTransactionReplacing{}, errors.New("error creating accumulating transactions, two transaction have different destination addresses")
		}

		if pt.PaymentSourceAddress != ref.PaymentSourceAddress {
			log.Print("Error creating accumulating transactions, two transactions have different source addresses")
			return PaymentTransactionReplacing{}, errors.New("error creating accumulating transactions, two transactions have different source addresses")
		}
	}

	pt.AmountOut = ref.AmountOut + pt.AmountOut

	transaction := PaymentTransactionReplacing{
		pendingTransaction:pt,
		referenceTransaction:ref,
	}

	return transaction,nil
}

func (payload *PaymentTransactionReplacing) GetPaymentTransaction() PaymentTransaction {
	return payload.pendingTransaction
}

func (payload *PaymentTransactionReplacing) Validate() error {
	return nil
}

func (payload *PaymentTransactionReplacing) GetPaymentDestinationAddress() string {
	return payload.pendingTransaction.PaymentDestinationAddress
}

func (payload *PaymentTransactionReplacing) UpdateTransactionXDR(xdr string) error {
	payload.pendingTransaction.XDR = xdr
	return nil
}

func (payload *PaymentTransactionReplacing) UpdateStellarToken(token string) error {
	payload.pendingTransaction.StellarNetworkToken = token
	return nil
}