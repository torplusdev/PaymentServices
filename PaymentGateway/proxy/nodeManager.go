package proxy

import (
	"paidpiper.com/payment-gateway/node"
)

type NodeManager struct {
	nodes map[string]node.PPNode
}

func New(localNode *node.Node) *NodeManager {
	manager := &NodeManager{
		make(map[string]node.PPNode),
	}

	manager.nodes[localNode.Address] = localNode

	return manager
}

func (m *NodeManager) GetNodeByAddress(address string) node.PPNode {
	n, ok := m.nodes[address]

	if ok {
		return n
	}

	return nil
}

func (m *NodeManager) AddNode(address string, node node.PPNode) {
	m.nodes[address] = node
}

func (m *NodeManager) GetProxyNode(address string) *NodeProxy {
	n, ok := m.nodes[address]

	if ok {
		return n.(*NodeProxy)
	}

	return nil
}
