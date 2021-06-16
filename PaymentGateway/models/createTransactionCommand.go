package models

type CreateTransactionCommand struct {
	TotalIn          uint32 `json:"TotalIn"`
	TotalOut         uint32 `json:"TotalOut"`
	SourceAddress    string `json:"SourceAddress"`
	ServiceSessionId string `json:"ServiceSessionId"`
}

func (cmd *CreateTransactionCommand) Type() CommandType {
	return CommandType_CreateTransaction
}

type CreateTransactionResponse struct {
	Transaction *PaymentTransactionReplacing `json:"Transaction"`
}

func (cmd *CreateTransactionResponse) Type() CommandType {
	return CommandType_CreateTransaction
}

func (cmd *CreateTransactionResponse) OutType() CommandType {
	return CommandType_CreateTransaction
}
