package node

import (
	"errors"
	"fmt"
	"paidpiper.com/payment-gateway/common"
	"runtime"
	"sync"
	"time"
)

type paymentRegistryEntry struct {
	mutex *sync.Mutex
	serviceSessionId string
	serviceNodeAddress string
	amount  common.TransactionAmount
	updated time.Time
}

type paymentRegistry struct {
	registryMutex 		*sync.Mutex
	ownAddress 			string
	entriesByAddress 	map[string]common.PaymentTransaction
	entriesBySessionId 	map[string]paymentRegistryEntry
	isActive bool
}

func createPaymentRegistry(ownAddress string) paymentRegistry {

	registry := paymentRegistry {
		registryMutex: &sync.Mutex{},
		ownAddress:ownAddress,
		entriesByAddress: make(map[string]common.PaymentTransaction),
		entriesBySessionId: make(map[string]paymentRegistryEntry),
		isActive: true,
	}

	go registry.performHousekeeping()

	return registry
}

func (r *paymentRegistry) performHousekeeping() {

	maxDurationBeforeExpiry,_ := time.ParseDuration("20s")
	sleepPeriod,_ := time.ParseDuration("3s")

	for {
		r.registryMutex.Lock()
		defer r.registryMutex.Unlock()

		for _,entry := range r.entriesBySessionId {
			entry.mutex.Lock()

			if (time.Since(entry.updated) >maxDurationBeforeExpiry) {
				delete(r.entriesBySessionId,entry.serviceSessionId)
			}

			entry.mutex.Unlock()
		}

		r.registryMutex.Unlock()
		time.Sleep(sleepPeriod)
		runtime.Gosched()
	}



}

func (r * paymentRegistry) getEntryBySessionId(serviceSessionId string) paymentRegistryEntry {
	r.registryMutex.Lock()
	defer r.registryMutex.Unlock()

	return r.entriesBySessionId[serviceSessionId]

}
func (r *paymentRegistry) AddServiceUsage(serviceSessionId string, amount common.TransactionAmount) {

	entry := r.getEntryBySessionId(serviceSessionId)

	if entry.updated.IsZero() {
		r.registryMutex.Lock()
		defer r.registryMutex.Unlock()

		if entry.updated.IsZero() {
			entry = paymentRegistryEntry{
				mutex:              &sync.Mutex{},
				serviceSessionId:   serviceSessionId,
				serviceNodeAddress: "",
				amount:             0,
				updated:            time.Now(),
			}
			r.entriesBySessionId[entry.serviceSessionId] = entry
		}
	}

	entry.mutex.Lock()
	entry.amount = entry.amount + amount
	entry.updated = time.Now()
	r.entriesBySessionId[entry.serviceSessionId] = entry
	entry.mutex.Unlock()
}

func (r *paymentRegistry) getPendingAmount(serviceSessionId string) (amount common.TransactionAmount,ok bool) {

	entry := r.getEntryBySessionId(serviceSessionId)

	if entry.updated.IsZero() {
		return 0, false
	} else {
		return entry.amount, true
	}

}

func (r *paymentRegistry) saveTransaction(paymentSourceAddress string, transaction *common.PaymentTransaction) {
	r.entriesByAddress[paymentSourceAddress] = *transaction
}

func (r *paymentRegistry) reducePendingAmount(serviceSessionId string, amount common.TransactionAmount) error {

	entry := r.getEntryBySessionId(serviceSessionId)

	if entry.updated.IsZero() {
		return errors.New(fmt.Sprintf("Specified serviceSessionId (%s) wasn't found",serviceSessionId))
	}

	entry.mutex.Lock()
	entry.amount = entry.amount - amount
	entry.updated = time.Now()
	r.entriesBySessionId[entry.serviceSessionId] = entry
	entry.mutex.Unlock()

	return nil
}

func (r *paymentRegistry) getActiveTransactions() []common.PaymentTransaction {
	// There is no lock here to prevent contention if this is called frequently, but it could be added
	transactions := make([]common.PaymentTransaction,0)

	for _,t := range r.entriesByAddress {
		transactions = append(transactions,t)
	}

	return transactions
}

func (r *paymentRegistry) completePayment(paymentSourceAddress string, serviceSessionId string) {
	delete(r.entriesByAddress, paymentSourceAddress)
}

func (r *paymentRegistry) getActiveTransaction(paymentSourceAddress string) common.PaymentTransaction {
	return r.entriesByAddress[paymentSourceAddress]
}

