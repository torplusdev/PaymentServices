package root

import (
	"fmt"
	"testing"
)

func TestAccountDetails(t *testing.T) {
	fmt.Println("TestAccountDetails")
	seed := "SDYDMWWMLZ4YOQOEBQY4EOQJE4JVHIUZRVEFN6DIVE5FIMIB6O3OHCAY"
	factory := CreateRootApiFactory(true)
	client, err := factory(seed, int64(400*1000))
	if err != nil {
		t.Errorf("CreateRootApiFactory Error:%v", err)
	}
	err = client.CheckSourceAddress("GC64BIJXPVUVN4OTJAF5Q4HCPFEWOEEBKKG57BQL76U3PVMYYEF3RLHW")
	if err != nil {

		t.Errorf("CheckSourceAddress Error:%v", err)
	}
}
