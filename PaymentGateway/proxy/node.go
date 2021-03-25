package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/go-errors/errors"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/api/trace"
	"io/ioutil"
	"log"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/models"
	"strconv"
	"sync"
)

type NodeProxy struct {
	mutex          *sync.Mutex
	address        string
	torUrl         string
	commandChannel map[string]chan []byte
	sessionId      string
	nodeId         string
	tracer         trace.Tracer
}

func (n NodeProxy) ProcessCommandNoReply(context context.Context, commandType int, commandBody string) error {
	id := uuid.New().String()

	values := map[string]string{"CommandId": id, "CommandType": strconv.Itoa(commandType), "CommandBody": commandBody, "NodeId": n.nodeId}

	jsonValue, _ := json.Marshal(values)

	res, err := common.HttpPostWithContext(context, n.torUrl, bytes.NewBuffer(jsonValue))
	defer res.Body.Close()

	return err
}

func (n NodeProxy) GetAddress() string {
	return n.address
}


/// We don't know (and don't need to) remote node balance,
func (n *NodeProxy) GetBalance() float64 {
	//TODO: Redesign the interface
	panic("GetBalance shouldn't be called on a remote node.")
	return 0;
}

func (n NodeProxy) ProcessCommand(context context.Context, commandType int, commandBody []byte) ([]byte, error) {
	id := uuid.New().String()

	command := &models.ProcessCommand{
		SessionId:   n.sessionId,
		NodeId:      n.nodeId,
		CommandId:   id,
		CommandType: commandType,
		CommandBody: commandBody,
	}

	log.Printf("Process command SessionId=%s, NodeId=%s, CommandId=%s CommandType:%d", n.sessionId, n.nodeId, id, commandType)

	jsonValue, _ := json.Marshal(command)

	ch := n.openCommandChannel(id)

	defer n.closeCommandChannel(id, ch)

	res, err := common.HttpPostWithoutContext(n.torUrl, bytes.NewBuffer(jsonValue))
	defer res.Body.Close()

	if err != nil {
		log.Fatal(err)

		return nil, err
	}

	bodyBytes, err := ioutil.ReadAll(res.Body)

	if err == nil && len(bodyBytes) > 0 {
		return bodyBytes, nil
	}

	if err != nil {
		log.Fatal(err)
	}

	// Wait
	responseBody := <-ch

	return responseBody, nil
}

func (n NodeProxy) openCommandChannel(id string) chan []byte {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	ch := make(chan []byte, 2)
	n.commandChannel[id] = ch

	return ch
}

func (n NodeProxy) closeCommandChannel(id string, ch chan []byte) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	delete(n.commandChannel, id)
	defer close(ch)
}

func (n NodeProxy) ProcessResponse(commandId string, responseBody []byte) {
	n.mutex.Lock()

	ch, ok := n.commandChannel[commandId]

	n.mutex.Unlock()

	if !ok {
		log.Printf("Unknown command response: : %s on %s", commandId, n.nodeId)
		return
	}

	ch <- responseBody
}

func (n NodeProxy) CreateTransaction(context context.Context, totalIn common.TransactionAmount, fee common.TransactionAmount, totalOut common.TransactionAmount, sourceAddress string, serviceSessionId string) (common.PaymentTransactionReplacing, error) {

	ctx, span := n.tracer.Start(context, "proxy-CreateTransaction-"+n.address)
	defer span.End()

	var request = &models.CreateTransactionCommand{
		TotalIn:          totalIn,
		TotalOut:         totalOut,
		SourceAddress:    sourceAddress,
		ServiceSessionId: serviceSessionId,
	}

	body, err := json.Marshal(request)

	if err != nil {
		return common.PaymentTransactionReplacing{}, err
	}

	reply, err := n.ProcessCommand(ctx, 0, body)

	if err != nil {
		return common.PaymentTransactionReplacing{}, err
	}

	response := &models.CreateTransactionResponse{}

	err = json.Unmarshal(reply, response)

	if err != nil {
		return common.PaymentTransactionReplacing{}, err
	}

	return response.Transaction, nil
}

func (n NodeProxy) SignTerminalTransactions(context context.Context, creditTransactionPayload *common.PaymentTransactionReplacing) error {

	ctx, span := n.tracer.Start(context, "proxy-SignTerminalTransactions-"+n.address)
	defer span.End()

	traceContext, err := common.CreateTraceContext(span.SpanContext())

	if err != nil {
		return err
	}

	var request = &models.SignTerminalTransactionCommand{
		Transaction: *creditTransactionPayload,
		Context:     traceContext,
	}

	body, err := json.Marshal(request)

	if err != nil {
		return err
	}

	reply, err := n.ProcessCommand(ctx, 1, body)

	if err != nil {
		return err
	}

	var response = &models.SignTerminalTransactionResponse{}

	err = json.Unmarshal(reply, response)

	if err != nil {
		return err
	}

	*creditTransactionPayload = response.Transaction

	return nil
}

func (n NodeProxy) SignChainTransactions(context context.Context, creditTransactionPayload *common.PaymentTransactionReplacing, debitTransactionPayload *common.PaymentTransactionReplacing) error {

	ctx, span := n.tracer.Start(context, "proxy-SignChainTransactions-"+n.address)
	defer span.End()

	traceContext, err := common.CreateTraceContext(span.SpanContext())

	if err != nil {
		return err
	}

	var request = &models.SignChainTransactionsCommand{
		Debit:   *debitTransactionPayload,
		Credit:  *creditTransactionPayload,
		Context: traceContext,
	}

	body, err := json.Marshal(request)

	if err != nil {
		return err
	}

	reply, err := n.ProcessCommand(ctx, 2, body)

	if err != nil {
		return err
	}

	var response = &models.SignChainTransactionsResponse{}

	err = json.Unmarshal(reply, response)

	if err != nil {
		return err
	}

	*creditTransactionPayload = response.Credit
	*debitTransactionPayload = response.Debit

	return nil
}

func (n NodeProxy) CommitServiceTransaction(context context.Context, transaction *common.PaymentTransactionReplacing, pr common.PaymentRequest) error {

	ctx, span := n.tracer.Start(context, "proxy-CommitServiceTransaction-"+n.address)
	defer span.End()

	traceContext, err := common.CreateTraceContext(span.SpanContext())

	if err != nil {
		return err
	}

	var request = &models.CommitServiceTransactionCommand{
		Transaction:    *transaction,
		PaymentRequest: pr,
		Context:        traceContext,
	}

	body, err := json.Marshal(request)

	if err != nil {
		return errors.Errorf(err.Error())
	}

	reply, err := n.ProcessCommand(ctx, 4, body)

	if err != nil {
		return errors.Errorf(err.Error())
	}

	var response = &models.CommitServiceTransactionResponse{}

	err = json.Unmarshal(reply, response)

	if err != nil {
		return errors.Errorf(err.Error())
	}

	return nil
}

func (n NodeProxy) CommitPaymentTransaction(context context.Context, transactionPayload *common.PaymentTransactionReplacing) error {

	ctx, span := n.tracer.Start(context, "proxy-CommitPaymentTransaction-"+n.address)
	defer span.End()

	traceContext, err := common.CreateTraceContext(span.SpanContext())

	if err != nil {
		return err
	}

	var request = &models.CommitPaymentTransactionCommand{
		Transaction: *transactionPayload,
		Context:     traceContext,
	}

	body, err := json.Marshal(request)

	if err != nil {
		return errors.Errorf(err.Error())
	}

	reply, err := n.ProcessCommand(ctx, 3, body)

	if err != nil {
		return errors.Errorf(err.Error())
	}

	var response = &models.CommitPaymentTransactionResponse{}

	err = json.Unmarshal(reply, response)

	if err != nil {
		return errors.Errorf(err.Error())
	}

	return nil
}
