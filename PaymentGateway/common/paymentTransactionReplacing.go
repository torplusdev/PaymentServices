package common

import (
	"errors"
	"fmt"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/trace"
	"log"
	"strconv"
)

type PaymentTransactionReplacing struct {
	PendingTransaction   PaymentTransaction
	ReferenceTransaction PaymentTransaction
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
	pt.ReferenceAmountIn = ref.ReferenceAmountIn + pt.ReferenceAmountIn

	transaction := PaymentTransactionReplacing{
		PendingTransaction:   pt,
		ReferenceTransaction: ref,
	}

	return transaction,nil
}

func (payload *PaymentTransactionReplacing) GetPaymentTransaction() *PaymentTransaction {
	return &payload.PendingTransaction
}

func (payload *PaymentTransactionReplacing) ToSpanAttributes(span trace.Span, transactionName string) {
	transaction := payload.PendingTransaction

	name := fmt.Sprintf("transaction.%s",transactionName)

	span.SetAttributes(core.KeyValue{ Key:core.Key(name + ".source-address"),Value: core.String(transaction.TransactionSourceAddress) })

	genericTransaction, _ := txnbuild.TransactionFromXDR(transaction.XDR)
	xdrTrans, result := genericTransaction.Transaction()

	if !result {
		span.SetAttributes(core.KeyValue{Key: core.Key(name + ".source-address"), Value: core.String("Error unpacking Transaction from GenericTransaction")})
		return
	}


	for i,op := range xdrTrans.Operations() {
		xdrOp, _ := op.BuildXDR()

		key := fmt.Sprintf("%s.xdr.op-%d-%s",name, i, xdrOp.Body.Type.String())

		span.SetAttributes(core.KeyValue{Key: core.Key(key), Value: core.String(xdrOp.Body.Type.String())})

		switch xdrOp.Body.Type {
		case xdr.OperationTypePayment:

			payment := &txnbuild.Payment{}

			err := payment.FromXDR(xdrOp)

			if err != nil {
				span.SetAttributes(core.KeyValue{Key: core.Key(key + ".error"), Value: core.String(err.Error())})
			}

			span.SetAttributes(core.KeyValue{Key: core.Key(key + ".from"), Value: core.String(payment.SourceAccount.GetAccountID())})
			span.SetAttributes(core.KeyValue{Key: core.Key(key + ".to"), Value: core.String(payment.Destination)})
			span.SetAttributes(core.KeyValue{Key: core.Key(key + ".amount"), Value: core.String(payment.Amount)})
			span.SetAttributes(core.KeyValue{Key: core.Key(key + ".asset"), Value: core.String(payment.Asset.GetCode())})
		default:
			span.SetAttributes(core.KeyValue{Key: core.Key(key + ".error"), Value: core.String("Unexpected operation type")})
		}
	}

	for i, signature := range xdrTrans.Signatures() {
		span.SetAttributes(core.KeyValue{ Key:core.Key(name + ".xdr.signature" + strconv.Itoa(i)),
			Value: core.String("Signature [" + strconv.Itoa(signature.Signature.XDRMaxSize()) + "]") })
	}

	span.SetAttributes(core.KeyValue{ Key:core.Key(name + ".xdr.sourceAccount"),Value: core.String(xdrTrans.SourceAccount().AccountID) })
	account := xdrTrans.SourceAccount()
	seq,_ := account.GetSequenceNumber()
	span.SetAttributes(core.KeyValue{ Key:core.Key(name + ".xdr.sequence"),Value: core.Int64(int64(seq)) })
}

func (payload *PaymentTransactionReplacing) validateSingleTransaction() error {

	activeTransaction := payload.PendingTransaction

	if (activeTransaction.ReferenceAmountIn < activeTransaction.AmountOut) {
		return errors.New("Transaction validation error: AmountOut cannot be larger than AmountIn")
	}

	if (activeTransaction.TransactionSourceAddress != activeTransaction.PaymentDestinationAddress) {
		return errors.New("Transaction validation error: should be sourced by the payment recepient")
	}

	if (activeTransaction.PaymentSourceAddress == activeTransaction.PaymentDestinationAddress) {
		return errors.New("Transaction validation error: Payer and payee cannot be the same address")
	}

	return nil
}

func (payload *PaymentTransactionReplacing) Validate() error {

	// Check that the transactions carry the same sequnce id
	payTrans := payload.GetPaymentTransaction()
	refTrans := payload.GetReferenceTransaction()

	payTransStellarWrapper,err := txnbuild.TransactionFromXDR(payTrans.XDR)

	if err != nil {
		return &TransactionValidationError{Source: "PaymentTransaction",
			Err: errors.New("validation error: couldn't deserialize payment transaction"),
		}
	}

	payTransStellar, result := payTransStellarWrapper.Transaction()

	if !result {
		return &TransactionValidationError{Source: "PaymentTransaction",
			Err: errors.New("validation error: couldn't deserialize payment transaction (GenericTransaction)"),
		}
	}

	err = payload.validateSingleTransaction()

	if err != nil {
		return &TransactionValidationError{Source: "TransactionPayload",
			Err: errors.New("validation error: error validating transaction: " + err.Error()),
		}
	}

	if refTrans != (PaymentTransaction{}) {
		refTransStellarWrapper,err := txnbuild.TransactionFromXDR(refTrans.XDR)

		if err != nil {
			return &TransactionValidationError{Source: "PaymentTransaction",
				Err: errors.New("validation error: couldn't deserialize reference transaction"),
			}
		}

		refTransStellar,result := refTransStellarWrapper.Transaction()

		if !result {
			return &TransactionValidationError{Source: "PaymentTransaction",
				Err: errors.New("validation error: couldn't deserialize reference transaction (GenericTransaction)"),
			}
		}

		account := payTransStellar.SourceAccount()
		paySequenceNumber,_ := account.GetSequenceNumber()

		account = refTransStellar.SourceAccount()
		refSequenceNumber,_ := account.GetSequenceNumber()

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
	return payload.PendingTransaction.PaymentDestinationAddress
}

func (payload *PaymentTransactionReplacing) UpdateTransactionXDR(xdr string) error {
	payload.PendingTransaction.XDR = xdr
	return nil
}

func (payload *PaymentTransactionReplacing) UpdateStellarToken(token string) error {
	payload.PendingTransaction.StellarNetworkToken = token
	return nil
}

func (payload *PaymentTransactionReplacing) GetReferenceTransaction() PaymentTransaction {
	return payload.ReferenceTransaction
}