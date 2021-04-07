package models

import (
	"fmt"
	"strconv"

	"github.com/go-errors/errors"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/trace"
)

type PaymentTransactionReplacing struct {
	PendingTransaction   PaymentTransaction
	ReferenceTransaction *PaymentTransaction
}

func (payload *PaymentTransactionReplacing) ToSpanAttributes(span trace.Span, transactionName string) {
	transaction := payload.PendingTransaction

	name := fmt.Sprintf("transaction.%s", transactionName)

	span.SetAttributes(core.KeyValue{Key: core.Key(name + ".source-address"), Value: core.String(transaction.TransactionSourceAddress)})

	genericTransaction, _ := transaction.XDR.TransactionFromXDR()
	xdrTrans, result := genericTransaction.Transaction()

	if !result {
		span.SetAttributes(core.KeyValue{Key: core.Key(name + ".source-address"), Value: core.String("Error unpacking Transaction from GenericTransaction")})
		return
	}

	for i, op := range xdrTrans.Operations() {
		xdrOp, _ := op.BuildXDR()

		key := fmt.Sprintf("%s.xdr.op-%d-%s", name, i, xdrOp.Body.Type.String())

		span.SetAttributes(core.KeyValue{Key: core.Key(key), Value: core.String(xdrOp.Body.Type.String())})

		switch xdrOp.Body.Type {
		case xdr.OperationTypePayment:

			payment := &txnbuild.Payment{}

			err := payment.FromXDR(xdrOp)

			if err != nil {
				span.SetAttributes(core.KeyValue{Key: core.Key(key + ".error"), Value: core.String(err.Error())})
			}

			span.SetAttributes(core.KeyValue{Key: core.Key(key + ".from"), Value: core.String(payment.SourceAccount)})
			span.SetAttributes(core.KeyValue{Key: core.Key(key + ".to"), Value: core.String(payment.Destination)})
			span.SetAttributes(core.KeyValue{Key: core.Key(key + ".amount"), Value: core.String(payment.Amount)})
			span.SetAttributes(core.KeyValue{Key: core.Key(key + ".asset"), Value: core.String(payment.Asset.GetCode())})
		default:
			span.SetAttributes(core.KeyValue{Key: core.Key(key + ".error"), Value: core.String("Unexpected operation type")})
		}
	}

	for i, signature := range xdrTrans.Signatures() {
		span.SetAttributes(core.KeyValue{Key: core.Key(name + ".xdr.signature" + strconv.Itoa(i)),
			Value: core.String("Signature [" + strconv.Itoa(signature.Signature.XDRMaxSize()) + "]")})
	}

	span.SetAttributes(core.KeyValue{Key: core.Key(name + ".xdr.sourceAccount"), Value: core.String(xdrTrans.SourceAccount().AccountID)})
	account := xdrTrans.SourceAccount()
	seq, _ := account.GetSequenceNumber()
	span.SetAttributes(core.KeyValue{Key: core.Key(name + ".xdr.sequence"), Value: core.Int64(int64(seq))})
}

func (payload *PaymentTransactionReplacing) GetPaymentDestinationAddress() string {
	return payload.PendingTransaction.PaymentDestinationAddress
}

func (payload *PaymentTransactionReplacing) Validate() error {

	// Check that the transactions carry the same sequnce id
	payTrans := payload.PendingTransaction
	refTrans := payload.ReferenceTransaction

	payTransStellarWrapper, err := payTrans.XDR.TransactionFromXDR()

	if err != nil {
		err = errors.New("validation error: couldn't deserialize payment transaction")
		return Wrap("PaymentTransaction", err)
	}

	payTransStellar, result := payTransStellarWrapper.Transaction()

	if !result {
		return Wrap("PaymentTransaction",
			errors.Errorf("validation error: couldn't deserialize payment transaction (GenericTransaction)"))
	}

	err = payTrans.validateSingleTransaction()

	if err != nil {
		return Wrap("TransactionPayload",
			errors.Errorf("validation error: error validating transaction: "+err.Error()))
	}

	if refTrans != nil {
		refTransStellarWrapper, err := refTrans.XDR.TransactionFromXDR()

		if err != nil {
			return Wrap("PaymentTransaction",
				errors.Errorf("validation error: couldn't deserialize reference transaction"))
		}

		refTransStellar, result := refTransStellarWrapper.Transaction()

		if !result {
			return Wrap("PaymentTransaction",
				errors.Errorf("validation error: couldn't deserialize reference transaction (GenericTransaction)"))
		}

		account := payTransStellar.SourceAccount()
		paySequenceNumber, _ := account.GetSequenceNumber()

		account = refTransStellar.SourceAccount()
		refSequenceNumber, _ := account.GetSequenceNumber()

		if paySequenceNumber != refSequenceNumber {
			return Wrap("TransactionPayload",
				errors.Errorf("validation error: different sequence numbers between transactions"))
		}

		//TODO: Check signatures
		//payTransStellar.TxEnvelope().Signatures[0]
	}

	return nil
}

func Wrap(source string, err error) error {
	return errors.Errorf("%v:%v", source, err)
}

func (payload *PaymentTransaction) validateSingleTransaction() error {

	activeTransaction := payload

	if activeTransaction.ReferenceAmountIn < activeTransaction.AmountOut {
		return errors.Errorf("transaction validation error: AmountOut cannot be larger than AmountIn")
	}

	if activeTransaction.TransactionSourceAddress != activeTransaction.PaymentDestinationAddress {
		return errors.Errorf("transaction validation error: should be sourced by the payment recepient")
	}

	if activeTransaction.PaymentSourceAddress == activeTransaction.PaymentDestinationAddress {
		return errors.Errorf("transaction validation error: Payer and payee cannot be the same address")
	}

	return nil
}
