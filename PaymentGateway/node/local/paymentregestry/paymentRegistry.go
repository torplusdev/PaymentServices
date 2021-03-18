package paymentregestry

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"paidpiper.com/payment-gateway/models"
)

type PaymentRegistry interface {
	AddServiceUsage(sourceAddress string, amount models.TransactionAmount)
	GetEntryByAddress(sourceAddress string) *paymentRegistryEntry
	GetTransactionBySessionId(sessionId string) *models.PaymentTransaction
	GetPendingAmount(sourceAddress string) (amount models.TransactionAmount, ok bool)
	SaveTransaction(paymentSourceAddress string, transaction *models.PaymentTransaction)
	ReducePendingAmount(sourceAddress string, amount models.TransactionAmount) error
	GetActiveTransactions() []*models.PaymentTransaction
	CompletePayment(paymentSourceAddress string, serviceSessionId string)
	GetActiveTransaction(paymentSourceAddress string) *models.PaymentTransaction
}

//-------
type paymentRegistry struct {
	mutex                       sync.Mutex
	paidTransactionsBySessionId map[string]*models.PaymentTransaction
	paidTransactionsByAddress   map[string]*models.PaymentTransaction
	entriesBySourceAddress      map[string]*paymentRegistryEntry
	isActive                    bool
	useHousekeeping             bool
}

func New() PaymentRegistry {

	registry := &paymentRegistry{
		mutex:                       sync.Mutex{},
		paidTransactionsBySessionId: make(map[string]*models.PaymentTransaction),
		paidTransactionsByAddress:   make(map[string]*models.PaymentTransaction),
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

	return r.paidTransactionsBySessionId[sessionId]
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

func (r *paymentRegistry) GetPendingAmount(sourceAddress string) (amount models.TransactionAmount, ok bool) {
	entry, ok := r.entriesBySourceAddress[sourceAddress]
	if ok {
		return entry.amount, true
	} else {
		return 0, false
	}
}

func (r *paymentRegistry) SaveTransaction(paymentSourceAddress string, transaction *models.PaymentTransaction) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.paidTransactionsByAddress[paymentSourceAddress] = transaction
	r.paidTransactionsBySessionId[transaction.ServiceSessionId] = transaction
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
	msg := fmt.Sprintf("Specified address (%s) wasn't found", sourceAddress)
	return fmt.Errorf(msg)

}

func (r *paymentRegistry) GetActiveTransactions() []*models.PaymentTransaction {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	tr := []*models.PaymentTransaction{}
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
	return r.paidTransactionsByAddress[paymentSourceAddress]
}
