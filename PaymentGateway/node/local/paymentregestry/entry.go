package paymentregestry

import (
	"sync"
	"time"

	"paidpiper.com/payment-gateway/models"
)

type paymentRegistryEntry struct {
	mutex              sync.Mutex
	serviceNodeAddress string
	amount             models.TransactionAmount
	updated            time.Time
}

func (pre *paymentRegistryEntry) since() time.Duration {
	pre.mutex.Lock()
	defer pre.mutex.Unlock()
	return time.Since(pre.updated)
}

func newEntry(sourceAddress string, amount models.TransactionAmount) *paymentRegistryEntry {
	return &paymentRegistryEntry{
		mutex:              sync.Mutex{},
		serviceNodeAddress: sourceAddress,
		amount:             amount,
		updated:            time.Now(),
	}
}

func (pre *paymentRegistryEntry) add(amount models.TransactionAmount) {
	pre.mutex.Lock()
	defer pre.mutex.Unlock()
	pre.updated = time.Now()
	pre.amount += amount
}

func (pre *paymentRegistryEntry) reduce(amount models.TransactionAmount) {
	pre.mutex.Lock()
	defer pre.mutex.Unlock()
	pre.updated = time.Now()
	pre.amount -= amount
}
