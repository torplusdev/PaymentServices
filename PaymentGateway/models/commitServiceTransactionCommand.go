package models

type CommitServiceTransactionCommand struct {
	Transaction    *PaymentTransactionReplacing `json:"transaction"`
	PaymentRequest *PaymentRequest              `json:"paymentRequest"`
	Context        *TraceContext                `json:"context"`
}

func (cmd *CommitServiceTransactionCommand) Type() CommandType {
	return CommandType_CommitServiceTransaction
}

type CommitServiceTransactionResponse struct {
}

func (cmd *CommitServiceTransactionResponse) OutType() CommandType {
	return CommandType_CommitServiceTransaction
}
