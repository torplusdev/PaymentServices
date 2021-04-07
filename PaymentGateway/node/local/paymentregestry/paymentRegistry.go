package paymentregestry

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"paidpiper.com/payment-gateway/models"
)

type PaymentRegistry interface {
	AddServiceUsage(sessionId string, amount models.TransactionAmount)
	ReducePendingAmount(sessionId string, amount models.TransactionAmount) error
	GetPendingAmount(sourceAddress string) (amount models.TransactionAmount, ok bool)
	SaveTransaction(sequence int64, transaction *models.PaymentTransaction)

	GetActiveTransactions() []*models.PaymentTransactionWithSequence
	CompletePayment(paymentSourceAddress string, serviceSessionId string)
	GetActiveTransaction(paymentSourceAddress string) *models.PaymentTransaction
	GetTransactionBySessionId(sessionId string) *models.PaymentTransaction
}

//-------
type paymentRegistry struct {
	mutex                       sync.Mutex
	paidTransactionsBySessionId map[string]*models.PaymentTransactionWithSequence
	paidTransactionsByAddress   map[string]*models.PaymentTransactionWithSequence
	entriesBySourceAddress      map[string]*paymentRegistryEntry
	isActive                    bool
	useHousekeeping             bool
}

func New() PaymentRegistry {

	registry := &paymentRegistry{
		mutex:                       sync.Mutex{},
		paidTransactionsBySessionId: make(map[string]*models.PaymentTransactionWithSequence),
		paidTransactionsByAddress:   make(map[string]*models.PaymentTransactionWithSequence),
		entriesBySourceAddress:      make(map[string]*paymentRegistryEntry),
		isActive:                    true,
	}
	if registry.useHousekeeping {
		go func(r *paymentRegistry) { //TODO DO add chain
			for range time.Tick(3 * time.Second) {
				r.cleanEvent()
				runtime.Gosched()
			}
		}(registry)
	}

	return registry
}

func (r *paymentRegistry) cleanEvent() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	maxDurationBeforeExpiry := 20 * time.Second
	for _, entry := range r.entriesBySourceAddress {

		if entry.since() > maxDurationBeforeExpiry {
			delete(r.entriesBySourceAddress, entry.serviceNodeAddress)
		}

	}

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

func (r *paymentRegistry) AddServiceUsage(sourceAddress string, amount models.TransactionAmount) {
	entry, ok := r.entriesBySourceAddress[sourceAddress]
	if ok {
		entry.add(amount)
	} else {
		entry = newEntry(sourceAddress, amount)
		r.entriesBySourceAddress[sourceAddress] = entry
	}
}

func (r *paymentRegistry) ReducePendingAmount(sourceAddress string, amount models.TransactionAmount) error {
	entry, ok := r.entriesBySourceAddress[sourceAddress]

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
