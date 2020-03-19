package common

type TransactionValidationError struct {
	Source string
	Err error
}

func (e *TransactionValidationError) Error() string { return e.Source + ": " + e.Err.Error() }

func (e *TransactionValidationError) Unwrap() error { return e.Err }

