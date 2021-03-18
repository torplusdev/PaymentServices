package tests

import (
	"context"

	"github.com/stellar/go/support/log"

	"github.com/stretchr/testify/assert"
	"paidpiper.com/payment-gateway/models"
)

type Sequencer struct {
	testSetup *TestSetup
	assert    *assert.Assertions
	ctx       context.Context
}

func CreateSequencer(testSetup *TestSetup, assert *assert.Assertions, ctx context.Context) Sequencer {
	sq := Sequencer{
		testSetup: testSetup,
		assert:    assert,
		ctx:       ctx,
	}

	return sq

}

func (sq Sequencer) PerformPayment(sourceSeed string, destinationSeed string, commodityAmount uint32) (*models.ProcessPaymentAccepted, *models.PaymentRequest) {

	pr, err := sq.testSetup.NewPaymentRequest(sq.ctx, destinationSeed, commodityAmount)
	sq.assert.NoError(err)

	if err != nil {
		log.Fatal("Error: " + err.Error())
	}

	result, err := sq.testSetup.ProcessPayment(sq.ctx, sourceSeed, pr)
	sq.assert.NoError(err)

	if err != nil {
		log.Fatal("Error: " + err.Error())
	}

	return result, pr

}
