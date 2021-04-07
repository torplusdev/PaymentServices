package proxy

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node"
)

type ProxyNode interface {
	node.PPNode

	ProcessResponse(context context.Context, commandId string, responseBody []byte) error
}

func NewProxyNode(commandClient CommandClient, responseHandler CommandResponseHandler, address string, fee uint32) ProxyNode {
	return &nodeProxy{
		commandClient:   commandClient,
		responseHandler: responseHandler,
		address:         address,
		tracer:          global.Tracer(fmt.Sprintf("nodeProxy-%v", address)),
		fee:             fee,
	}
}

type nodeProxy struct {
	commandClient   CommandClient
	responseHandler CommandResponseHandler
	address         string
	tracer          trace.Tracer
	fee             uint32
}

func (n *nodeProxy) GetFee() uint32 {
	return n.fee
}

func (n *nodeProxy) GetAddress() string {
	return n.address
}

func (n *nodeProxy) ProcessResponse(context context.Context, commandId string, responseBody []byte) error {
	return n.responseHandler.ProcessResponse(context, commandId, responseBody)
}

func (n *nodeProxy) CreateTransaction(context context.Context, command *models.CreateTransactionCommand) (*models.CreateTransactionResponse, error) {

	ctx, span := n.tracer.Start(context, "proxy-CreateTransaction-"+n.address)
	defer span.End()
	//TODO CHECK
	return n.commandClient.CreateTransaction(ctx, command)

}

func (n *nodeProxy) SignServiceTransaction(context context.Context, command *models.SignServiceTransactionCommand) (*models.SignServiceTransactionResponse, error) {

	ctx, span := n.tracer.Start(context, "proxy-SignServiceTransaction-"+n.address)
	defer span.End()

	traceContext, err := models.NewTraceContext(span.SpanContext())
	if err != nil {
		return nil, err
	}
	command.Context = traceContext

	return n.commandClient.SignServiceTransaction(ctx, command)
}

func (n *nodeProxy) SignChainTransaction(context context.Context, command *models.SignChainTransactionCommand) (*models.SignChainTransactionResponse, error) {

	ctx, span := n.tracer.Start(context, "proxy-SignChainTransaction-"+n.address)
	defer span.End()
	traceContext, err := models.NewTraceContext(span.SpanContext())
	if err != nil {
		return nil, err
	}
	command.Context = traceContext
	return n.commandClient.SignChainTransaction(ctx, command)
}

func (n *nodeProxy) CommitServiceTransaction(context context.Context, command *models.CommitServiceTransactionCommand) error {

	ctx, span := n.tracer.Start(context, "proxy-CommitServiceTransaction-"+n.address)
	defer span.End()

	traceContext, err := models.NewTraceContext(span.SpanContext())
	if err != nil {
		return err
	}
	command.Context = traceContext
	return n.commandClient.CommitServiceTransaction(ctx, command)
}

func (n *nodeProxy) CommitChainTransaction(context context.Context, command *models.CommitChainTransactionCommand) error {
	ctx, span := n.tracer.Start(context, "proxy-CommitChainTransaction-"+n.address)
	defer span.End()

	traceContext, err := models.NewTraceContext(span.SpanContext())
	if err != nil {
		return err
	}
	command.Context = traceContext
	return n.commandClient.CommitChainTransaction(ctx, command)
}
