package paymentregestry

import (
	"database/sql"
	"time"

	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node/local/paymentregestry/database"
	"paidpiper.com/payment-gateway/node/local/paymentregestry/database/entity"
)

type paymentRegistryWithDb struct {
	PaymentRegistry
	db database.Db
}

func NewWithDB(db database.Db) (PaymentRegistry, error) {
	pr, err := New()
	if err != nil {
		return nil, err
	}

	regestryWithDb := &paymentRegistryWithDb{
		PaymentRegistry: pr,
		db:              db,
	}

	return regestryWithDb, nil
}
func (prdb *paymentRegistryWithDb) LogError(err error) {
	//TODO TO LOGGER
}

func (prdb *paymentRegistryWithDb) AddServiceUsage(sessionId string, pr *models.PaymentRequest) {
	err := prdb.db.InsertPaymentRequest(&entity.DbPaymentRequest{
		SessionId:        sessionId,
		Amount:           int(pr.Amount),
		Asset:            pr.Asset,
		Address:          pr.Address,
		ServiceRef:       pr.ServiceRef,
		ServiceSessionId: pr.ServiceSessionId,
		Date:             time.Now(),
		CompleteDate:     sql.NullTime{},
	})
	if err != nil {
		prdb.LogError(err)
	}
	prdb.PaymentRegistry.AddServiceUsage(sessionId, pr)
}

func (prdb *paymentRegistryWithDb) ReducePendingAmount(sessionId string, amount models.TransactionAmount) error {

	return prdb.PaymentRegistry.ReducePendingAmount(sessionId, amount)
}

func (prdb *paymentRegistryWithDb) GetPendingAmount(sourceAddress string) (amount models.TransactionAmount, ok bool) {

	return prdb.PaymentRegistry.GetPendingAmount(sourceAddress)
}

func (prdb *paymentRegistryWithDb) SaveTransaction(sequence int64, transaction *models.PaymentTransaction) {
	err := prdb.db.InsertTransaction(&entity.DbTransactoin{
		Sequence: sequence,
	})
	if err != nil {
		prdb.LogError(err)
	}
	prdb.PaymentRegistry.SaveTransaction(sequence, transaction)
}

func (prdb *paymentRegistryWithDb) GetActiveTransactions() []*models.PaymentTransactionWithSequence {

	return prdb.PaymentRegistry.GetActiveTransactions()
}

func (prdb *paymentRegistryWithDb) CompletePayment(paymentSourceAddress string, serviceSessionId string) {

	prdb.PaymentRegistry.CompletePayment(paymentSourceAddress, serviceSessionId)

}

func (prdb *paymentRegistryWithDb) GetActiveTransaction(paymentSourceAddress string) *models.PaymentTransaction {
	return prdb.PaymentRegistry.GetActiveTransaction(paymentSourceAddress)
}

func (prdb *paymentRegistryWithDb) GetTransactionBySessionId(sessionId string) *models.PaymentTransaction {

	return prdb.PaymentRegistry.GetTransactionBySessionId(sessionId)
}
