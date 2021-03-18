package paymentmanager

import "paidpiper.com/payment-gateway/models"

type Debt struct {
	id models.PeerID

	requestedAmount uint32

	transferredBytes uint32

	receivedBytes uint32
}
type DebtRegestry interface {
	GetDebt(id models.PeerID) *Debt
}
type debtRegestryImpl struct {
	store map[models.PeerID]*Debt
}

func (r *debtRegestryImpl) GetDebt(id models.PeerID) *Debt {
	debt, ok := r.store[id]

	if ok {
		return debt
	}

	debt = &Debt{
		id: id,
	}

	r.store[id] = debt

	return debt
}
