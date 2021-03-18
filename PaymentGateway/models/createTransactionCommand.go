package models

type CreateTransactionCommand struct {
	TotalIn          uint32 `json:"totalIn"`
	TotalOut         uint32 `json:"totalOut"`
	SourceAddress    string `json:"sourceAddress"`
	ServiceSessionId string `json:"serviceSessionId"`
}

func (cmd *CreateTransactionCommand) Type() CommandType {
	return CommandType_CreateTransaction
}

type CreateTransactionResponse struct {
	Transaction *PaymentTransactionReplacing `json:"transaction"`
}

func (cmd *CreateTransactionResponse) Type() CommandType {
	return CommandType_CreateTransaction
}

func (cmd *CreateTransactionResponse) OutType() CommandType {
	return CommandType_CreateTransaction
}
