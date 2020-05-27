package integration_tests

import (
	"context"
	"github.com/stretchr/testify/assert"
	"paidpiper.com/payment-gateway/common"
)

type sequencer struct {
	testSetup *TestSetup
	assert *assert.Assertions
	ctx context.Context
}

func createSequencer(testSetup *TestSetup, assert *assert.Assertions, ctx context.Context) sequencer {
	sq := sequencer{
		testSetup: testSetup,
		assert:    assert,
		ctx:       ctx,
	}

	return sq

}

func (sq sequencer) performPayment(sourceSeed string, destinationSeed string, paymentAmount float64) (string,common.PaymentRequest) {

	pr,err := sq.testSetup.CreatePaymentInfo(sq.ctx, destinationSeed,int(paymentAmount))
	sq.assert.NoError(err)



	result, err := sq.testSetup.ProcessPayment(sq.ctx, sourceSeed,pr)
	sq.assert.NoError(err)

	return result,pr

}