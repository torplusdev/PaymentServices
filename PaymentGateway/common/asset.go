package common

import (
	"math"
	"strconv"
)

const PPTokenIssuerAddress = "GC3NJX52DCCY6B6ABJ2236O7J2F5XBATNUT3VS2T6BNBOTP7T4X3KFCX"
const PPTokenAssetName = "pptoken"
const PPTokenUnitPrice = 1e-6 // uPP

func PPTokenToString(amount TransactionAmount) string {
	return strconv.FormatFloat(PPTokenUnitPrice*float64(amount),'f', 7,64)
	//return strconv.FormatUint(uint64(amount), 10)
}

func MicroPPToken2PPtoken(micro float64 ) TransactionAmount {
	return TransactionAmount(math.Round(micro/PPTokenUnitPrice))
}

func PPtoken2MicroPP(pptoken TransactionAmount )  float64 {
	return PPTokenUnitPrice * float64(pptoken)
}

type PPTokenAsset struct {
}

//func (token *PPTokenAsset) GetPaymentTransaction() *PaymentTransaction {
//
//}
