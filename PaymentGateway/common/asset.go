package common

import (
	"strconv"
)

const PPTokenIssuerAddress = "GCW3GHZEZCKR5QAXYSLJ6PB2Y2VUMQ75VKJNYCSTEFDNRQHJFF3U65IY"
const PPTokenAssetName = "pptoken"
const PPTokenUnitPrice = 1e-7

func PPTokenToString(amount TransactionAmount) string {
	return strconv.FormatUint(uint64(amount), 10)
}

type PPTokenAsset struct {
}

//func (token *PPTokenAsset) GetPaymentTransaction() *PaymentTransaction {
//
//}
