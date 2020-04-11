package models

import "paidpiper.com/payment-gateway/common"

type UtilityCommand struct {
	CommandType int		`json:"commandType"`
	CommandBody	string	`json:"commandBody"`
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

type GetStellarAddressResponse struct {
	Address	string
}