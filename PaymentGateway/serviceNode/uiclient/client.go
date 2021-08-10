package uiclient

import (
	webUiClient "paidpiper.com/provider-service/pkg/client"

	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/node/local"
)

type uiClient struct {
	nodeNode local.LocalPPNode
}

func New(nodeNode local.LocalPPNode) webUiClient.Client {
	return &uiClient{nodeNode: nodeNode}
}

func (c *uiClient) GetAddress() (string, error) {
	return c.nodeNode.GetAddress(), nil
}

func (c *uiClient) GetBalance() (float64, error) {
	bal, err := c.nodeNode.GetBookBalance()
	if err != nil {
		return 0, err
	}
	return bal.Balance, nil
}

func (c *uiClient) GetTransactions(limits int32) (interface{}, error) {
	trs, err := c.nodeNode.GetTransactionHistory(common.DirectionCredit, int(limits))
	if err != nil {
		return nil, err
	}
	return trs.Items, nil
}

func (c *uiClient) GetChartData(hours int32) (interface{}, error) {
	trs, err := c.nodeNode.GetTransactionHistoryGroup(int(hours))
	if err != nil {
		return nil, err
	}
	return trs.Items, nil
}

func (c *uiClient) GetSyncInfo() (interface{}, error) {
	return c.nodeNode.GetTransactionInfo(), nil
}
