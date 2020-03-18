package common

import "github.com/stellar/go/xdr"

type StaticSequenceProvider struct {
	sequenceId uint64
}

func CreateStaticSequence(sequence uint64) StaticSequenceProvider {
	return StaticSequenceProvider{
		sequenceId:sequence,
	}
}

func (seq StaticSequenceProvider) SequenceForAccount(aid string) (xdr.SequenceNumber, error) {
	return xdr.SequenceNumber(seq.sequenceId),nil
}
