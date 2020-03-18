package proxy

import (
	"encoding/json"
	"net/http"
	"paidpiper.com/payment-gateway/controllers"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node"
)

type NodeManager struct {
	nodes map[string]NodeProxy
}

func (m *NodeManager) GetNodeByAddress(id string) node.PPNode {
	return m.nodes[id]
}

func (m *NodeManager) ProcessResponse(w http.ResponseWriter, r *http.Request) {
	response := &models.UtilityResponse{}

	err := json.NewDecoder(r.Body).Decode(response)

	if err != nil {
		controllers.Respond(w, controllers.Message(false, "Invalid request"))
		return
	}

	proxy := m.nodes[response.NodeId]

	proxy.ProcessResponse(response.CommandId, response.ResponseBody)
}