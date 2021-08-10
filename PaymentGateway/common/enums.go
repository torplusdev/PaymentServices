package common

import (
	"errors"
	"fmt"
	"strings"
)

type TransactionDirection int

const (
	DirectionCredit TransactionDirection = 0 // Credit
	DirectionDebit  TransactionDirection = 1 // Debit
)

var directionText = map[int]string{
	int(DirectionCredit): "Credit",
	int(DirectionDebit):  "Debit",
}

func (tr TransactionDirection) IsValid() bool {
	switch tr {
	case DirectionCredit, DirectionDebit:
		return true
	}
	return false
}

func IsDirectionValid(value string) bool {
	for _, d := range directionText {
		if strings.EqualFold(value, d) {
			return true
		}
	}
	return false
}

func GetDirectionByString(str string) (TransactionDirection, error) {
	for idx, d := range directionText {
		if strings.EqualFold(str, d) {
			return TransactionDirection(idx), nil
		}
	}
	fmt.Println()
	return -1, errors.New("String is not valid TransactionDirection.")
}

func DirectionText(idx int) string {
	return directionText[idx]
}
