package commodity

import (
	"errors"
	"fmt"
)

type Descriptor struct {
	UnitPrice	float64
	Asset		string
}

type Manager struct {
	priceTable	map[string]map[string]Descriptor

}

func New(priceTable	map[string]map[string]Descriptor) *Manager {
	return &Manager{priceTable:priceTable}
}

func (cm *Manager) Calculate(service string, commodity string, quantity uint32) (price uint32, asset string, err error) {
	st, ok := cm.priceTable[service]

	if !ok {
		return 0, "", errors.New(fmt.Sprintf("unknown service %s", service))
	}

	d, ok := st[commodity]

	if !ok {
		return 0, "", errors.New(fmt.Sprintf("unknown commodity %s", commodity))
	}

	return uint32(d.UnitPrice * float64(quantity)), d.Asset, nil
}

func (cm *Manager) ReverseCalculate(service string, commodity string, price uint32, asset string) (quantity uint32, err error) {
	st, ok := cm.priceTable[service]

	if !ok {
		return 0, errors.New(fmt.Sprintf("unknown service %s", service))
	}

	d, ok := st[commodity]

	if !ok {
		return 0, errors.New(fmt.Sprintf("unknown commodity %s", commodity))
	}

	if d.Asset != asset {
		return 0, errors.New(fmt.Sprintf("asset missmatch %s", asset))
	}

	return uint32(float64(price) / d.UnitPrice), nil
}
