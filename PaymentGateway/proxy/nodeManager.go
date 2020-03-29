package proxy

import (
	"paidpiper.com/payment-gateway/node"
)

type NodeManager struct {
	nodes map[string]node.PPNode
	torUrl string
}

func New(localNode *node.Node, torUrl string) *NodeManager {
	manager := &NodeManager{
		make(map[string]node.PPNode),
		torUrl,
	}

	manager.nodes[localNode.Address] = localNode

	return manager
}

func (m *NodeManager) GetNodeByAddress(address string) node.PPNode {
	n, ok := m.nodes[address]

	if ok {
		return n
	}

	n = NewProxy(address, m.torUrl)

	m.nodes[address] = n

	return n
}

func (m *NodeManager) GetProxyNode(address string) NodeProxy {
	return m.nodes[address].(NodeProxy)
}
