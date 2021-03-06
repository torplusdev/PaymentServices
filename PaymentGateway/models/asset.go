package models

import (
	"math"
	"strconv"
)

const PPTokenIssuerAddress = "GCW3GHZEZCKR5QAXYSLJ6PB2Y2VUMQ75VKJNYCSTEFDNRQHJFF3U65IY"
const PPTokenAssetName = "pptoken"
const PPTokenMinAllowedBalance = 10
const PPTokenUnitPrice = 1e-3 // uPP

func PPTokenToString(amount TransactionAmount) string {
	return strconv.FormatFloat(PPTokenUnitPrice*float64(amount), 'f', 7, 64)
	//return strconv.FormatUint(uint64(amount), 10)
}

func PPTokenToNumeric(transactionAmount float64) float64 {
	return PPTokenUnitPrice * transactionAmount
}

func MicroPPToken2PPtoken(micro float64) TransactionAmount {
	return TransactionAmount(math.Round(micro / PPTokenUnitPrice))
}

func PPtoken2MicroPP(pptoken TransactionAmount) float64 {
	return PPTokenUnitPrice * float64(pptoken)
}

type PPTokenAsset struct {
}

//func (token *PPTokenAsset) GetPendingTransaction() *PaymentTransaction {
//
//}
