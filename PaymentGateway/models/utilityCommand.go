package models

import (
	"paidpiper.com/payment-gateway/common"
	"time"
)

type UtilityCommand struct {
	SessionId   string `json:"sessionId"`
	NodeId      string `json:"nodeId"`
	CommandId   string `json:"commandId"`
	CommandType int    `json:"commandType"`
	CommandBody []byte `json:"commandBody"`
	CallbackUrl string `json:"callbackUrl"`
}

type CreateTransactionCommand struct {
	TotalIn          uint32 `json:"totalIn"`
	TotalOut         uint32 `json:"totalOut"`
	SourceAddress    string `json:"sourceAddress"`
	ServiceSessionId string `json:"serviceSessionId"`
}

type CreateTransactionResponse struct {
	Transaction common.PaymentTransactionReplacing `json:"transaction"`
}

type SignTerminalTransactionCommand struct {
	Transaction common.PaymentTransactionReplacing `json:"transaction"`
	Context     common.TraceContext                `json:"context"`
}

type SignTerminalTransactionResponse struct {
	Transaction common.PaymentTransactionReplacing `json:"transaction"`
}

type SignChainTransactionsCommand struct {
	Debit   common.PaymentTransactionReplacing `json:"debit"`
	Credit  common.PaymentTransactionReplacing `json:"credit"`
	Context common.TraceContext                `json:"context"`
}

type SignChainTransactionsResponse struct {
	Debit  common.PaymentTransactionReplacing `json:"debit"`
	Credit common.PaymentTransactionReplacing `json:"credit"`
}

type CommitPaymentTransactionCommand struct {
	Transaction common.PaymentTransactionReplacing `json:"transaction"`
	Context     common.TraceContext                `json:"context"`
}

type CommitPaymentTransactionResponse struct {

}

type CommitServiceTransactionCommand struct {
	Transaction    common.PaymentTransactionReplacing `json:"transaction"`
	PaymentRequest common.PaymentRequest              `json:"paymentRequest"`
	Context        common.TraceContext                `json:"context"`
}

type CommitServiceTransactionResponse struct {

}

type GetStellarAddressResponse struct {
	Address string
}

type GetBalanceResponse struct {
	Balance float64
	Timestamp time.Time
}
type GetPendingPaymentResponse struct {
	Address        string
	PendingBalance common.TransactionAmount
	Timestamp      time.Time
}
