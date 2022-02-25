package paymentregestry

import (
	"fmt"
	"sync"

	"paidpiper.com/payment-gateway/models"
)

type PaymentRegistry interface {
	AddServiceUsage(sourceAddress string, pr *models.PaymentRequest)
	AddServiceСonsumption(sessionId string, pr *models.PaymentRequest)
	ReducePendingAmount(sourceAddress string, amount models.TransactionAmount) error
	GetPendingAmount(sourceAddress string) (amount models.TransactionAmount, ok bool)

	SaveTransaction(sequence int64, transaction *models.PaymentTransaction)
	CompletePayment(paymentSourceAddress string, serviceSessionId string)

	GetActiveTransactions() []*models.PaymentTransactionWithSequence
	GetActiveTransaction(paymentSourceAddress string) *models.PaymentTransaction
	GetTransactionBySessionId(sessionId string) *models.PaymentTransaction
}

//-------
type paymentRegistry struct {
	mutex                       sync.Mutex
	paidTransactionsBySessionId map[string]*models.PaymentTransactionWithSequence
	paidTransactionsByAddress   map[string]*models.PaymentTransactionWithSequence
	entriesBySourceAddress      map[string]*paymentRegistryEntry
	//isActive                    bool
	//useHousekeeping bool
}

func New() (PaymentRegistry, error) {

	registry := &paymentRegistry{
		mutex:                       sync.Mutex{},
		paidTransactionsBySessionId: make(map[string]*models.PaymentTransactionWithSequence),
		paidTransactionsByAddress:   make(map[string]*models.PaymentTransactionWithSequence),
		entriesBySourceAddress:      make(map[string]*paymentRegistryEntry),
		//isActive:                    true,
	}
	// if registry.useHousekeeping { //TODO CHECK NEED OR NOT
	// 	go func(r *paymentRegistry) { //TODO DO add chain
	// 		for range time.Tick(3 * time.Second) {
	// 			r.cleanEvent()
	// 			runtime.Gosched()
	// 		}
	// 	}(registry)
	// }

	return registry, nil
}

// func (r *paymentRegistry) cleanEvent() { //TODO CHECK NEED OR NOT
// 	r.mutex.Lock()
// 	defer r.mutex.Unlock()
// 	maxDurationBeforeExpiry := 20 * time.Second
// 	for _, entry := range r.entriesBySourceAddress {
// 		if entry.since() > maxDurationBeforeExpiry {
// 			delete(r.entriesBySourceAddress, entry.serviceNodeAddress)
// 		}
// 	}

// }

func (r *paymentRegistry) AddServiceСonsumption(sessionId string, pr *models.PaymentRequest) {

}

func (r *paymentRegistry) GetEntryByAddress(sourceAddress string) *paymentRegistryEntry {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.entriesBySourceAddress[sourceAddress]
}

func (r *paymentRegistry) GetTransactionBySessionId(sessionId string) *models.PaymentTransaction {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	pts, ok := r.paidTransactionsBySessionId[sessionId]
	if ok {
		return &pts.PaymentTransaction
	}
	return nil
}

func (r *paymentRegistry) AddServiceUsage(sourceAddress string, pr *models.PaymentRequest) {
	entry, ok := r.entriesBySourceAddress[sourceAddress]
	if ok {
		entry.add(pr.Amount)
	} else {
		entry = newEntry(sourceAddress, pr.Amount)
		r.entriesBySourceAddress[sourceAddress] = entry
	}
}

func (r *paymentRegistry) ReducePendingAmount(sourceAddress string, amount models.TransactionAmount) error {
	entry, ok := r.entriesBySourceAddress[sourceAddress] //TODO is by source id

	if ok {
		entry.reduce(amount)
		if entry.amount == 0 {
			r.mutex.Lock()
			defer r.mutex.Unlock()
			delete(r.entriesBySourceAddress, sourceAddress)
		}
		return nil
	}
	return fmt.Errorf("specified address (%s) wasn't found", sourceAddress)
}

func (r *paymentRegistry) GetPendingAmount(sourceAddress string) (amount models.TransactionAmount, ok bool) {
	entry, ok := r.entriesBySourceAddress[sourceAddress]
	if ok {
		return entry.amount, true
	} else {
		return 0, false
	}
}

func (r *paymentRegistry) SaveTransaction(sequence int64, transaction *models.PaymentTransaction) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	tr := &models.PaymentTransactionWithSequence{
		PaymentTransaction: *transaction,
		Sequence:           sequence,
	}
	r.paidTransactionsByAddress[transaction.PaymentSourceAddress] = tr
	r.paidTransactionsBySessionId[transaction.ServiceSessionId] = tr
}

func (r *paymentRegistry) GetActiveTransactions() []*models.PaymentTransactionWithSequence {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	tr := []*models.PaymentTransactionWithSequence{}
	for _, t := range r.paidTransactionsByAddress {
		tr = append(tr, t)
	}
	return tr
}

func (r *paymentRegistry) CompletePayment(paymentSourceAddress string, serviceSessionId string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	delete(r.paidTransactionsByAddress, paymentSourceAddress)
	delete(r.paidTransactionsBySessionId, serviceSessionId)
}

func (r *paymentRegistry) GetActiveTransaction(paymentSourceAddress string) *models.PaymentTransaction {
	item, ok := r.paidTransactionsByAddress[paymentSourceAddress]
	if ok {
		return &item.PaymentTransaction
	}
	return nil
}
