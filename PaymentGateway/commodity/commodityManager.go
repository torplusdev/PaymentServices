package commodity

import (
	"errors"
	"fmt"
)

type Descriptor struct {
	Name 		string
	UnitPrice	uint32
	Asset		string
}

type Manager struct {
	priceTable	map[string]Descriptor

}

func New(priceTable	map[string]Descriptor) *Manager {
	return &Manager{priceTable:priceTable}
}

func (cm *Manager) Calculate(name string, quantity uint32) (price uint32, asset string, err error) {
	d, ok := cm.priceTable[name]

	if !ok {
		return 0, "", errors.New(fmt.Sprintf("unknown commodity %s", name))
	}

	return d.UnitPrice * quantity, d.Asset, nil
}
