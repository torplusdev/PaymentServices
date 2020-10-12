package common

import (
	"strconv"
)

const PPTokenIssuerAddress = "GC3NJX52DCCY6B6ABJ2236O7J2F5XBATNUT3VS2T6BNBOTP7T4X3KFCX"
const PPTokenAssetName = "pptoken"
const PPTokenMinAllowedBalance = 10
const PPTokenUnitPrice = 1e-8

func PPTokenToString(amount TransactionAmount) string {
	return strconv.FormatUint(uint64(amount), 10)
}

type PPTokenAsset struct {
}

//func (token *PPTokenAsset) GetPaymentTransaction() *PaymentTransaction {
//
//}
