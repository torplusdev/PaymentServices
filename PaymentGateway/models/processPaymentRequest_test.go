package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestErrorFormat(t *testing.T) {
	err := errors.New("err ms")
	parentErr := fmt.Errorf("Parent error: %v", err)
	log := parentErr.Error()
	if log != "Parent error: err ms" {
		t.Error("Invalid formatting")
	}
}

func TestMarshalTest(t *testing.T) {
	m := &ProcessPaymentRequest{}
	bs, err := json.Marshal(m)
	if err != nil {
		t.Error(err)
	}
	//serString := string(bs)
	//fmt.Println(serString)
	//t.Log(serString)

	unm := &ProcessPaymentRequest{}
	err = json.Unmarshal(bs, unm)
	if err != nil {
		t.Error(err)
	}
	if !cmp.Equal(*unm, *m) {
		t.Error("Not equal")
	}

}

func TestUnmarsalEmptyStruct(t *testing.T) {

	err := json.Unmarshal([]byte("{   }"), &struct{}{})
	if err != nil {
		t.Error(err)
	} else {
		t.Log("Unmarshal success")
	}
}
