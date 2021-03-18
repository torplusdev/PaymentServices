package models

type SignServiceTransactionCommand struct {
	Transaction *PaymentTransactionReplacing `json:"transaction"`
	Context     *TraceContext                `json:"context"`
}

func (cmd *SignServiceTransactionCommand) Type() CommandType {
	return CommandType_SignServiceTransaction
}

type SignServiceTransactionResponse struct {
	Transaction *PaymentTransactionReplacing `json:"transaction"`
}

func (cmd *SignServiceTransactionResponse) OutType() CommandType {
	return CommandType_SignServiceTransaction
}
