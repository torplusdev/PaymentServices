package proxy

import (
	"go.opentelemetry.io/otel/api/global"
	"paidpiper.com/payment-gateway/node"
)

type NodeManager struct {
	nodesByAddress map[string]node.PPNode
	nodesByNodeId map[string]node.PPNode
}

func New(localNode *node.Node) *NodeManager {
	manager := &NodeManager{
		nodesByAddress: make(map[string]node.PPNode),
		nodesByNodeId: make(map[string]node.PPNode),
	}

	manager.nodesByAddress[localNode.Address] = localNode

	return manager
}

func (m *NodeManager) AddNode(address string, nodeId string, torUrl string) {
	n := &NodeProxy{
		address:        address,
		torUrl:         torUrl,
		commandChannel: make(map[string]chan string),
		nodeId:         nodeId,
		tracer:         global.Tracer("nodeProxy"),
	}

	m.nodesByAddress[address] = n
	m.nodesByNodeId[nodeId] = n
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
