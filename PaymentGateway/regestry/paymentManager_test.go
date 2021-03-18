package regestry

import (
	"testing"
)

func TestPaymentManager(t *testing.T) {
	NewNodeManager()
}

// func FromSeeds(useTestApi bool, seeds []string) (PaymentManager, error) {
// 	rootClientFactory := root.CreateRootApiFactory(useTestApi)
// 	clientRootApi, err := rootClientFactory(seeds[0], 600)
// 	paymentClient := client.New(clientRootApi)
// 	if err != nil {
// 		return nil, err
// 	}
// 	paymentRequst := &models.ProcessPaymentRequest{}
// 	pm := NewPaymentManager(paymentClient, paymentRequst, "")
// 	i := 0
// 	for _, seed := range seeds {
// 		client, err := rootClientFactory(seed, 600)
// 		nodeId := xid.New().String()
// 		if err != nil {
// 			return nil, err
// 		}
// 		if i == 0 {
// 			localNode, err := local.New(client, nil, true, 3*time.Second)
// 			if err != nil {
// 				return nil, err
// 			}
// 			pm.AddSourceNode(client.GetAddress(), localNode)
// 		}
// 		if i == 1 {
// 			localNode, err := local.New(client, nil, true, 3*time.Second)
// 			if err != nil {
// 				return nil, err
// 			}
// 			pm.AddChainNode(client.GetAddress(), nodeId, localNode)
// 		}
// 		if i == 1 {
// 			localNode, err := local.New(client, nil, true, 3*time.Second)
// 			if err != nil {
// 				return nil, err
// 			}
// 			pm.AddDestinationNode(client.GetAddress(), nodeId, localNode)
// 		}

// 	}

// 	return pm, nil
// }
