package common

import (
	"errors"
	"github.com/stellar/go/txnbuild"
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

func (payload *PaymentTransactionReplacing) GetPaymentTransaction() *PaymentTransaction {
	return &payload.pendingTransaction
}

func (payload *PaymentTransactionReplacing) validateSingleTransaction() error {

	//TODO: Add transaction validation

	return nil
}

func (payload *PaymentTransactionReplacing) Validate() error {

	// Check that the transactions carry the same sequnce id
	payTrans := payload.GetPaymentTransaction()
	refTrans := payload.GetReferenceTransaction()

	payTransStellar,err := txnbuild.TransactionFromXDR(payTrans.XDR)

	if err != nil {
		return &TransactionValidationError{Source: "PaymentTransaction",
			Err: errors.New("validation error: couldn't deserialize payment transaction"),
		}
	}

	err = payload.validateSingleTransaction()

	if err != nil {
		return &TransactionValidationError{Source: "TransactionPayload",
			Err: errors.New("validation error: error validating transaction: " + err.Error()),
		}
	}

	if refTrans != (PaymentTransaction{}) {
		refTransStellar,err := txnbuild.TransactionFromXDR(refTrans.XDR)

		if err != nil {
			return &TransactionValidationError{Source: "PaymentTransaction",
				Err: errors.New("validation error: couldn't deserialize reference transaction"),
			}
		}

		paySequenceNumber,_ := payTransStellar.SourceAccount.(*txnbuild.SimpleAccount).GetSequenceNumber()
		refSequenceNumber,_ := refTransStellar.SourceAccount.(*txnbuild.SimpleAccount).GetSequenceNumber()

		if paySequenceNumber != refSequenceNumber {
			return &TransactionValidationError{Source: "TransactionPayload",
				Err: errors.New("validation error: different sequence numbers between transactions"),
			}
		}

		//TODO: Check signatures
		//payTransStellar.TxEnvelope().Signatures[0]
	}

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

func (payload *PaymentTransactionReplacing) GetReferenceTransaction() PaymentTransaction {
	return payload.referenceTransaction
}