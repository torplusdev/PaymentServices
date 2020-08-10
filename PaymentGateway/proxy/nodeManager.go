package proxy

import (
	"errors"
	"go.opentelemetry.io/otel/api/global"
	"paidpiper.com/payment-gateway/node"
	"sync"
)

type NodeManager struct {
	mutex          *sync.Mutex
	nodesByAddress map[string]node.PPNode
	nodesByNodeId  map[string]node.PPNode
}

func New(localNode node.PPNode) *NodeManager {
	manager := &NodeManager{
		mutex:          &sync.Mutex{},
		nodesByAddress: make(map[string]node.PPNode),
		nodesByNodeId:  make(map[string]node.PPNode),
	}

	manager.nodesByAddress[localNode.GetAddress()] = localNode

	return manager
}

func (m *NodeManager) AddNode(address string, nodeId string, torUrl string, sessionId string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	n1, ok1 := m.nodesByAddress[address]
	n2, ok2 := m.nodesByNodeId[nodeId]

	if ok1 && ok2 {

		if n1 != n2 {
			return errors.New("address mapped to different node id")
		}

		return nil
	}

	if !ok1 && !ok2 {
		n := &NodeProxy{
			mutex:          &sync.Mutex{},
			address:        address,
			torUrl:         torUrl,
			commandChannel: make(map[string]chan []byte),
			nodeId:         nodeId,
			tracer:         global.Tracer("nodeProxy"),
			sessionId:      sessionId,
		}

		m.nodesByAddress[address] = n
		m.nodesByNodeId[nodeId] = n

		return nil
	}

	return errors.New("address in use for different node id")
}

func (m *NodeManager) GetNodeByAddress(address string) node.PPNode {
	n, ok := m.nodesByAddress[address]

	if ok {
		return n
	}

	return nil
}

func (m *NodeManager) GetProxyNode(nodeId string) *NodeProxy {
	n, ok := m.nodesByNodeId[nodeId]

	if ok {
		return n.(*NodeProxy)
	}

	return nil
}
