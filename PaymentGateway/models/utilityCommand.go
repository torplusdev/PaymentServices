package models

import "paidpiper.com/payment-gateway/common"

type UtilityCommand struct {
	CommandId	string 	`json:"commandId"`
	CommandType int		`json:"commandType"`
	CommandBody	string	`json:"commandBody"`
	NodeId      string  `json:"nodeId"`
}

type CreateTransactionCommand struct {
	TotalIn       uint32	`json:"totalIn"`
	TotalOut      uint32	`json:"totalOut"`
	SourceAddress string	`json:"sourceAddress"`
}

type CreateTransactionResponse struct {
	Transaction common.PaymentTransactionReplacing	`json:"transaction"`
}

type SignTerminalTransactionCommand struct {
	Transaction common.PaymentTransactionReplacing	`json:"transaction"`
}

type SignTerminalTransactionResponse struct {
	Transaction common.PaymentTransactionReplacing	`json:"transaction"`
}

type SignChainTransactionsCommand struct {
	Debit  common.PaymentTransactionReplacing	`json:"debit"`
	Credit common.PaymentTransactionReplacing	`json:"credit"`
}

type SignChainTransactionsResponse struct {
	Debit  common.PaymentTransactionReplacing	`json:"debit"`
	Credit common.PaymentTransactionReplacing	`json:"credit"`
}

type CommitPaymentTransactionCommand struct {
	Transaction common.PaymentTransactionReplacing	`json:"transaction"`
}

type CommitPaymentTransactionResponse struct {
	Ok bool	`json:"ok"`
}

type CommitServiceTransactionCommand struct {
	Transaction common.PaymentTransactionReplacing	`json:"transaction"`
	PaymentRequest common.PaymentRequest			`json:"paymentRequest"`
}

type CommitServiceTransactionResponse struct {
	Ok bool	`json:"ok"`
}

type CreatePaymentRequestCommand struct {
	ServiceSessionId string	`json:"serviceSessionId"`
}

type CreatePaymentRequestResponse struct {
	PaymentRequest common.PaymentRequest	`json:"paymentRequest"`
}

type AddPendingServicePaymentCommand struct {
	ServiceSessionId string 					`json:"serviceSessionId"`
	Amount           common.TransactionAmount	`json:"amount"`
}
