package testutils

import (
	"paidpiper.com/payment-gateway/node"
)

type TestNodeManager struct {
	nodes map[string]*node.Node
}

func CreateTestNodeManager() *TestNodeManager {

	nm := TestNodeManager {
		nodes: make(map[string]*node.Node),
	}

	return &nm
}

func (nm *TestNodeManager) AddNode(node *node.Node) (*TestNodeManager) {

	nm.nodes[node.Address] = node
	return nm
}

func (nm *TestNodeManager) GetNodeByAddress(address string) (*node.Node) {

	return nm.nodes[address]
}
