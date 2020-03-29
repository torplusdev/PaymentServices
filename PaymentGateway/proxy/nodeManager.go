package proxy

import (
	"encoding/json"
	"net/http"
	"paidpiper.com/payment-gateway/controllers"
	"paidpiper.com/payment-gateway/models"
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
	node, ok := m.nodes[address]

	if ok {
		return node
	}

	node = NewProxy(address, m.torUrl)

	m.nodes[address] = node

	return node
}

func (m *NodeManager) ProcessResponse(w http.ResponseWriter, r *http.Request) {
	response := &models.UtilityResponse{}

	err := json.NewDecoder(r.Body).Decode(response)

	if err != nil {
		controllers.Respond(500, w, controllers.Message("Invalid request"))
		return
	}

	proxy := m.nodes[response.NodeId].(NodeProxy)

	proxy.ProcessResponse(response.CommandId, response.ResponseBody)
}