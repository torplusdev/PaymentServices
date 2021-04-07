package models

//TODO
type SignChainTransactionCommand struct {
	Debit   *PaymentTransactionReplacing `json:"debit"`
	Credit  *PaymentTransactionReplacing `json:"credit"`
	Context *TraceContext                `json:"context"`
}

func (cmd *SignChainTransactionCommand) Type() CommandType {
	return CommandType_SignChainTransaction
}

type SignChainTransactionResponse struct {
	Debit  *PaymentTransactionReplacing `json:"debit"`
	Credit *PaymentTransactionReplacing `json:"credit"`
}

func (cmd *SignChainTransactionResponse) OutType() CommandType {
	return CommandType_SignChainTransaction
}
