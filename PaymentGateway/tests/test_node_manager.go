package testutils

import (
	"paidpiper.com/payment-gateway/node"
)

type TestNodeManager struct {
	nodes map[string]node.PPNode
}

func CreateTestNodeManager() *TestNodeManager {
	nm := TestNodeManager {
		nodes: make(map[string]node.PPNode),
	}

	return &nm
}

func (nm *TestNodeManager) AddNode(node node.PPNode) *TestNodeManager {

	nm.nodes[node.GetAddress()] = node
	return nm
}

func (nm *TestNodeManager) GetNodeByAddress(address string) node.PPNode {

	return nm.nodes[address]
}

func (nm *TestNodeManager) SetAccumulatingTransactionsMode(newMode bool) *TestNodeManager {

	for _,n := range nm.nodes {
		n.SetAccumulatingTransactionsMode(newMode)
	}

	return nm
}

// Replaces the node with the provided address with the supplied implementation.
func (nm *TestNodeManager) ReplaceNode(address string, newNode node.PPNode) *TestNodeManager {

	if nm.nodes[address] == nil {
		panic("Address doesnt exist")
	}
	nm.nodes[address] = newNode

	return nm
}



