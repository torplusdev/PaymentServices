package models

type CommitChainTransactionCommand struct {
	Transaction *PaymentTransactionReplacing `json:"transaction"`
	Context     *TraceContext                `json:"context"`
}

func (cmd *CommitChainTransactionCommand) Type() CommandType {
	return CommandType_CommitChainTransaction
}

type CommitChainTransactionResponse struct {
}

func (cmd *CommitChainTransactionResponse) OutType() CommandType {
	return CommandType_CommitChainTransaction
}
