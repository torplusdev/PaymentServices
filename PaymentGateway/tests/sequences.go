package tests

import (
	"context"
	"log"

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

func (sq Sequencer) PerformPayment(sourceSeed string, destinationSeed string, commodityAmount uint32) (*models.ProcessPaymentAccepted, *models.PaymentRequest, error) {

	pr, err := sq.testSetup.NewPaymentRequest(sq.ctx, destinationSeed, commodityAmount)
	sq.assert.NoError(err)

	if err != nil {
		log.Fatalf("Error: %v", err)
		return nil, nil, err
	}

	result, err := sq.testSetup.ProcessPayment(sq.ctx, sourceSeed, pr)
	sq.assert.NoError(err)

	if err != nil {
		log.Fatalf("Error: %v", err)
		return nil, nil, err
	}

	return result, pr, nil

}
