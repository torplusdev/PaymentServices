package models

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestUnmarshal(t *testing.T) {
	ut := &UtilityCommand{
		CommandCore: CommandCore{
			SessionId:   "",
			NodeId:      "",
			CommandId:   "",
			CommandType: 0,
		},
		CommandBody: &CreateTransactionCommand{
			TotalIn:          0,
			TotalOut:         0,
			SourceAddress:    "SourceAddress",
			ServiceSessionId: "ServiceSessionId",
		},

		CallbackUrl: "",
	}
	bs, err := json.Marshal(ut)
	if err != nil {
		t.Error(err)
	}
	marshalStr := string(bs)
	fmt.Println(marshalStr)

	unm := &UtilityCommand{}
	err = json.Unmarshal(bs, unm)
	if err != nil {
		t.Error(err)
	}

	if !cmp.Equal(ut, unm) {
		t.Errorf("Not equals")
	}
}
