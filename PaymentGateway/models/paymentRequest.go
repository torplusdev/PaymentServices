package models

type TransactionAmount = uint32
type PeerID string

func (st *PeerID) String() string {
	return string(*st)
}

type PaymentRequstBase struct {
	Amount     TransactionAmount
	Asset      string
	ServiceRef string
}
type PaymentRequest struct {
	Amount           TransactionAmount
	Asset            string
	ServiceRef       string
	ServiceSessionId string
	Address          string
}

/*

type inPaymentRequest PaymentRequest

func (pr *PaymentRequest) MarshalJSON() (bs []byte, err error) {
	inPr := inPaymentRequest(*pr)
	bs, err = json.Marshal(inPr)
	if err != nil {
		return
	}

	prStr, err := json.Marshal(string(bs))
	if err != nil {
		return
	}
	bs = []byte(prStr)
	return
}

func (pr *PaymentRequest) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	b = []byte(s)
	inPr := &inPaymentRequest{}
	err = json.Unmarshal(b, inPr)
	if err != nil {
		return err
	}
	pr.Address = inPr.Address
	pr.Amount = inPr.Amount
	pr.Asset = inPr.Asset
	pr.ServiceRef = inPr.ServiceRef
	pr.ServiceSessionId = inPr.ServiceSessionId
	return nil
}
*/
