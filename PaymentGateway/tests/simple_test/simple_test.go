package simple_test_test

import (
	"github.com/stellar/go/keypair"
	"testing"
)

func TestCreateSimpleKey(t *testing.T) {

	kp,_ := keypair.Random()

	address := kp.Address()
	seed := kp.Seed()
	a := seed

	_ = a
	_ = address

}