package regestry

import (
	"fmt"
	"strings"
	"sync"

	"paidpiper.com/payment-gateway/node"
)

type NodeManager interface {
	AddSourceNode(address string, node node.PPNode) error
	AddChainNode(address string, node node.PPNode) error
	AddDestinationNode(address string, node node.PPNode) error
	Has(address string) bool
	GetNodeByAddress(address string) node.PPNode
	GetSourceNode() node.PPNode
	GetDestinationNode() node.PPNode
	Validate(from string, to string) error
	GetAllNodes() []node.PPNode
}
type nodeManager struct {
	mutex           *sync.Mutex
	nodesByAddress  map[string]node.PPNode
	nodesBySequence []node.PPNode
	chaincomplete   bool
}

func NewNodeManager() NodeManager {
	return &nodeManager{
		mutex:          &sync.Mutex{},
		nodesByAddress: make(map[string]node.PPNode),
	}
}

func (m *nodeManager) Has(address string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	_, ok := m.nodesByAddress[address]
	return ok
}

func (m *nodeManager) AddSourceNode(address string, node node.PPNode) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if len(m.nodesBySequence) > 0 {
		return fmt.Errorf("source node already exists")
	}
	return m.addNode(address, node)
}

func (m *nodeManager) AddChainNode(address string, node node.PPNode) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.chaincomplete {
		return fmt.Errorf("chain already exists")
	}
	return m.addNode(address, node)
}

func (m *nodeManager) AddDestinationNode(address string, node node.PPNode) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.chaincomplete {
		return fmt.Errorf("destination node already exists")
	}
	err := m.addNode(address, node)
	if err != nil {
		return err
	}
	m.chaincomplete = true
	return nil

}

func (m *nodeManager) Validate(from string, to string) error {

	//validate route extremities
	firstItem := m.nodesBySequence[0]
	if strings.Compare(firstItem.GetAddress(), from) != 0 {
		return fmt.Errorf("bad routing: Incorrect starting address %s != %s", firstItem.GetAddress(), from)
	}
	lastItem := m.nodesBySequence[len(m.nodesBySequence)-1]
	if strings.Compare(lastItem.GetAddress(), to) != 0 {
		return fmt.Errorf("incorrect destination address, %s != %s", lastItem.GetAddress(), to)
	}
	return nil
}

func (m *nodeManager) GetSourceNode() node.PPNode {
	return m.nodesBySequence[0]
}

func (m *nodeManager) GetDestinationNode() node.PPNode {
	return m.nodesBySequence[len(m.nodesBySequence)-1]
}

func (m *nodeManager) GetAllNodes() []node.PPNode {
	return []node.PPNode(m.nodesBySequence)
}

func (m *nodeManager) addNode(address string, node node.PPNode) error {
	_, ok := m.nodesByAddress[address]
	if ok {
		return fmt.Errorf("node already exists")
	}
	m.nodesByAddress[address] = node
	m.nodesBySequence = append(m.nodesBySequence, node)
	return nil
}

func (m *nodeManager) GetNodeByAddress(address string) node.PPNode {
	n, ok := m.nodesByAddress[address]

	if ok {
		return n
	}

	return nil
}
