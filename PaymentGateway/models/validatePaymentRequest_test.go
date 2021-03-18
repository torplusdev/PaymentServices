package models

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

func TestMarshalValidatePaymentRequest(t *testing.T) {
	assert := assert.New(t)
	m := &ValidatePaymentRequest{
		PaymentRequest: PaymentRequest{},
	}
	bs, err := json.Marshal(m)
	assert.Nil(err)

	// serString := string(bs)
	// fmt.Println(serString)
	// t.Log(serString)

	unm := &ValidatePaymentRequest{}
	err = json.Unmarshal(bs, unm)
	assert.Nil(err)

	if !cmp.Equal(*unm, *m) {
		t.Error("Not equal")
	}

}
