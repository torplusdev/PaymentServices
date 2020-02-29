package common


type PaymentTransactionSimple struct {
	pendingTransaction PaymentTransaction
}

func CreateSimpleTransaction (pt PaymentTransaction) (*PaymentTransactionSimple) {

	transaction := PaymentTransactionSimple{
		pendingTransaction:pt,
	}

	return &transaction
}

func (payload *PaymentTransactionSimple) GetPaymentTransaction() PaymentTransaction {
	return payload.pendingTransaction
}

func (payload *PaymentTransactionSimple) Validate() error {
	return nil
}

func (payload *PaymentTransactionSimple) GetPaymentDestinationAddress() string {
	return payload.pendingTransaction.PaymentDestinationAddress
}

func (payload *PaymentTransactionSimple) UpdateTransactionXDR(xdr string) error {
	payload.pendingTransaction.XDR = xdr
	return nil
}

func (payload *PaymentTransactionSimple) UpdateStellarToken(token string) error {
	payload.pendingTransaction.StellarNetworkToken = token
	return nil
}
