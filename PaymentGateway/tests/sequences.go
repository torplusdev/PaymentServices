package tests

import (
	"context"
	"github.com/stellar/go/support/log"
	"github.com/stretchr/testify/assert"
	"paidpiper.com/payment-gateway/common"
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

func (sq Sequencer) PerformPayment(sourceSeed string, destinationSeed string, paymentAmount float64) (string, common.PaymentRequest) {

	pr, err := sq.testSetup.CreatePaymentInfo(sq.ctx, destinationSeed, int(paymentAmount))
	sq.assert.NoError(err)

	if err != nil {
		log.Error("Error: " + err.Error())
	}


	result, err := sq.testSetup.ProcessPayment(sq.ctx, sourceSeed, pr)
	sq.assert.NoError(err)

	if err != nil {
		log.Error("Error: " + err.Error())
	}

	return result, pr

}
