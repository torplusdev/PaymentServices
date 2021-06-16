package models

import (
	"encoding/json"

	"github.com/go-errors/errors"
	"github.com/stellar/go/txnbuild"
)

type XDR interface {
	Validate() error
	TransactionFromXDR() (*GenericTransaction, error)
	Empty() bool
	Equals(o XDR) bool
	String() string
}
type GenericTransaction struct {
	txnbuild.GenericTransaction
}

// func (g GenericTransaction) Transaction() (*Transaction, bool) {
// 	tr, res := g.GenericTransaction.Transaction()

// 	return &Transaction{
// 		in: *tr,
// 	}, res
// }

// type Transaction struct {
// 	in txnbuild.Transaction
// }

func NewXDR(xdr string) XDR {
	x := implXDR(xdr)
	return &x
}

type implXDR string

func (xdr *implXDR) Equals(o XDR) bool {
	return xdr.String() == o.String()
}

func (xdr *implXDR) Empty() bool {
	return len(xdr.String()) == 0
}

func (xdr *implXDR) String() string {
	return string(*xdr)
}

func (xdr *implXDR) Validate() error {
	_, err := txnbuild.TransactionFromXDR(string(*xdr))
	return err
}

func (xdr *implXDR) TransactionFromXDR() (*GenericTransaction, error) {
	tr, err := txnbuild.TransactionFromXDR(string(*xdr))
	if err != nil {
		return nil, err
	}
	return &GenericTransaction{
		GenericTransaction: *tr,
	}, nil

}

type PaymentTransactionWithSequence struct {
	PaymentTransaction
	Sequence int64
}
type PaymentTransaction struct {
	TransactionSourceAddress  string
	ReferenceAmountIn         TransactionAmount
	AmountOut                 TransactionAmount
	XDR                       XDR `json:"-"`
	PaymentSourceAddress      string
	PaymentDestinationAddress string
	StellarNetworkToken       string
	ServiceSessionId          string
}

func (d PaymentTransaction) MarshalJSON() ([]byte, error) {
	type InPaymentTransaction PaymentTransaction
	var typ struct {
		InPaymentTransaction
		XDR string
	}
	typ.InPaymentTransaction = InPaymentTransaction(d)
	typ.XDR = d.XDR.String()
	return json.Marshal(typ)
}

func (d *PaymentTransaction) UnmarshalJSON(b []byte) error {
	type InPaymentTransaction PaymentTransaction
	var typ struct {
		InPaymentTransaction
		XDR string
	}
	err := json.Unmarshal(b, &typ)
	if err != nil {
		return err
	}
	typ.InPaymentTransaction.XDR = NewXDR(typ.XDR)
	*d = PaymentTransaction(typ.InPaymentTransaction)
	return nil
}

func (pt *PaymentTransaction) Validate() error {
	if pt.PaymentSourceAddress == pt.PaymentDestinationAddress {
		return errors.Errorf("error invalid transaction chain, address targets itself %s.", pt.PaymentSourceAddress)
	}
	return nil
}

type PaymentTransactionPayload interface {
	GetPaymentTransaction() PaymentTransaction
	GetPaymentDestinationAddress() string
	UpdateTransactionXDR(xdr string) error
	UpdateStellarToken(token string) error
	Validate() error
}

type PaymentNode struct {
	Address string
	Fee     TransactionAmount
}

type PaymentRouter interface {
	CreatePaymentRoute(req *PaymentRequest) []PaymentNode
	GetNodeByAddress(address string) (PaymentNode, error)
}
