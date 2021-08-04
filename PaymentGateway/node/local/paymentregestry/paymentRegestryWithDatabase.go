package paymentregestry

import (
	"database/sql"
	"time"

	"paidpiper.com/payment-gateway/log"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node/local/paymentregestry/database"
	"paidpiper.com/payment-gateway/node/local/paymentregestry/database/dbtime"
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

func (prdb *paymentRegistryWithDb) openDb() bool {
	err := prdb.db.Open()
	if err != nil {
		prdb.LogError(err)
		return false
	}
	return true
}
func (prdb *paymentRegistryWithDb) closeDb() {
	err := prdb.db.Close()
	if err != nil {
		prdb.LogError(err)
	}
}
func (prdb *paymentRegistryWithDb) LogError(err error) {
	//TODO TO LOGGER
	log.Errorf("Payment regestry error: %s", err)
}

func (prdb *paymentRegistryWithDb) AddServiceUsage(sessionId string, pr *models.PaymentRequest) {
	if prdb.openDb() {
		defer prdb.closeDb()
		err := prdb.db.InsertPaymentRequest(&entity.DbPaymentRequest{
			SessionId:        sessionId,
			Amount:           int(pr.Amount),
			Asset:            pr.Asset,
			Address:          pr.Address,
			ServiceRef:       pr.ServiceRef,
			ServiceSessionId: pr.ServiceSessionId,
			Date:             time.Now(),
			CompleteDate:     sql.NullTime{Valid: false},
		})
		if err != nil {
			prdb.LogError(err)
		}
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
	if prdb.openDb() {
		defer prdb.closeDb()
		err := prdb.db.InsertTransaction(&entity.DbTransaction{
			Sequence: sequence,

			TransactionSourceAddress:  transaction.TransactionSourceAddress,
			ReferenceAmountIn:         int(transaction.ReferenceAmountIn),
			AmountOut:                 int(transaction.AmountOut),
			XDR:                       transaction.XDR.String(),
			PaymentSourceAddress:      transaction.PaymentSourceAddress,
			PaymentDestinationAddress: transaction.PaymentDestinationAddress,
			StellarNetworkToken:       transaction.StellarNetworkToken,
			ServiceSessionId:          transaction.ServiceSessionId,
			Date:                      dbtime.Now(),
		})
		if err != nil {
			prdb.LogError(err)
		}
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
