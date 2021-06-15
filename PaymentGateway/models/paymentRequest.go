package models

type TransactionAmount = uint32
type PeerID string

func (st *PeerID) String() string {
	return string(*st)
}

type PaymentRequstBase struct {
	Amount     TransactionAmount
	Asset      string
	ServiceRef string
}
type PaymentRequest struct {
	Amount           TransactionAmount
	Asset            string
	ServiceRef       string
	ServiceSessionId string
	Address          string
}
