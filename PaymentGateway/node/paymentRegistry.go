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
	serviceNodeAddress string
	amount  common.TransactionAmount
	updated time.Time
}

type paymentRegistry struct {
	registryMutex 					*sync.Mutex
	ownAddress 						string
	paidTransactionsBySessionId 	map[string]common.PaymentTransaction
	paidTransactionsByAddress 		map[string]common.PaymentTransaction
	entriesBySourceAddress 			map[string]paymentRegistryEntry
	isActive 						bool
	useHouskeeping 					bool
}

func createPaymentRegistry(ownAddress string) paymentRegistry {

	registry := paymentRegistry {
		registryMutex: &sync.Mutex{},
		ownAddress:ownAddress,
		paidTransactionsBySessionId: 	make(map[string]common.PaymentTransaction),
		paidTransactionsByAddress: 		make(map[string]common.PaymentTransaction),
		entriesBySourceAddress: 		make(map[string]paymentRegistryEntry),
		isActive: true,
	}

	go registry.performHousekeeping()

	return registry
}

func (r *paymentRegistry) performHousekeeping() {

	maxDurationBeforeExpiry,_ := time.ParseDuration("20s")
	sleepPeriod,_ := time.ParseDuration("3s")

	if !r.useHouskeeping {
		return
	}

	for {
		r.registryMutex.Lock()
		defer r.registryMutex.Unlock()

		for _,entry := range r.entriesBySourceAddress {
			entry.mutex.Lock()

			if (time.Since(entry.updated) >maxDurationBeforeExpiry) {
				delete(r.entriesBySourceAddress,entry.serviceNodeAddress)
			}

			entry.mutex.Unlock()
		}

		r.registryMutex.Unlock()
		time.Sleep(sleepPeriod)
		runtime.Gosched()
	}
}

func (r * paymentRegistry) getEntryByAddress(sourceAddress string) paymentRegistryEntry {
	r.registryMutex.Lock()
	defer r.registryMutex.Unlock()

	return r.entriesBySourceAddress[sourceAddress]
}

func (r *paymentRegistry) AddServiceUsage(sourceAddress string, amount common.TransactionAmount) {

	entry := r.getEntryByAddress(sourceAddress)

	if entry.updated.IsZero() {
		r.registryMutex.Lock()
		defer r.registryMutex.Unlock()

		if entry.updated.IsZero() {
			entry = paymentRegistryEntry{
				mutex:              &sync.Mutex{},
				serviceNodeAddress: sourceAddress,
				amount:             0,
				updated:            time.Now(),
			}
			r.entriesBySourceAddress[sourceAddress] = entry
		}
	}

	entry.mutex.Lock()

	entry.amount = entry.amount + amount
	entry.updated = time.Now()
	r.entriesBySourceAddress[sourceAddress] = entry

	entry.mutex.Unlock()
}

func (r *paymentRegistry) getPendingAmount(sourceAddress string) (amount common.TransactionAmount,ok bool) {

	entry := r.getEntryByAddress(sourceAddress)

	if entry.updated.IsZero() {
		return 0, false
	} else {
		return entry.amount, true
	}

}

func (r *paymentRegistry) saveTransaction(paymentSourceAddress string, transaction *common.PaymentTransaction) {
	r.paidTransactionsByAddress[paymentSourceAddress] 	= *transaction
	r.paidTransactionsBySessionId[transaction.ServiceSessionId] = *transaction
}

func (r *paymentRegistry) reducePendingAmount(sourceAddress string, amount common.TransactionAmount) error {

	entry := r.getEntryByAddress(sourceAddress)

	if entry.updated.IsZero() {
		return errors.New(fmt.Sprintf("Specified address (%s) wasn't found",sourceAddress))
	}

	entry.mutex.Lock()
	entry.amount = entry.amount - amount
	entry.updated = time.Now()
	r.entriesBySourceAddress[sourceAddress] = entry

	if (entry.amount == 0) {
		delete(r.entriesBySourceAddress, sourceAddress)
	}

	entry.mutex.Unlock()

	return nil
}

func (r *paymentRegistry) getActiveTransactions() []common.PaymentTransaction {
	// There is no lock here to prevent contention if this is called frequently, but it could be added
	transactions := make([]common.PaymentTransaction,0)

	for _,t := range r.paidTransactionsByAddress {
		transactions = append(transactions,t)
	}

	return transactions
}

func (r *paymentRegistry) completePayment(paymentSourceAddress string, serviceSessionId string) {
	r.registryMutex.Lock()
	defer r.registryMutex.Unlock()
	delete(r.paidTransactionsByAddress, paymentSourceAddress)
	delete(r.paidTransactionsBySessionId, serviceSessionId)
}

func (r *paymentRegistry) getActiveTransaction(paymentSourceAddress string) common.PaymentTransaction {
	return r.paidTransactionsByAddress[paymentSourceAddress]
}

