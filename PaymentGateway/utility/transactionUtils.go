package utility

import (
	"github.com/go-errors/errors"
	"github.com/stellar/go/txnbuild"
	"paidpiper.com/payment-gateway/common"
)

func PaymentTransactionToStellar(trans *common.PaymentTransaction) (*txnbuild.Transaction, error) {

	transactionWrapper, err := txnbuild.TransactionFromXDR(trans.XDR)

	if err != nil {
		return nil, errors.Errorf("Error converting transaction from xdr: %s",err.Error());
	}

	actualTransaction, result := transactionWrapper.Transaction()

	if !result {
		return nil, errors.Errorf("Error converting transaction i from xdr (GenericTransaction): %s",result);
	}

	return actualTransaction, nil
}