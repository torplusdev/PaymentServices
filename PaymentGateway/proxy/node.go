package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/go-errors/errors"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/api/trace"
	"log"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/models"
	testutils "paidpiper.com/payment-gateway/tests"
	"strconv"
)

type NodeProxy struct {
	address		       string
	torUrl         string
	commandChannel map[string]chan string
	nodeId      string
	tracer 		   trace.Tracer
}

func (n NodeProxy) ProcessCommandNoReply(context context.Context, commandType int, commandBody string) error {
	id := uuid.New().String()

	values := map[string]string{"CommandId": id, "CommandType": strconv.Itoa(commandType), "CommandBody": commandBody, "NodeId": n.nodeId}

	jsonValue, _ := json.Marshal(values)

	_, err := common.HttpPostWithContext(context,n.torUrl,bytes.NewBuffer(jsonValue))

	return err
}

func (n NodeProxy) ProcessCommand(context context.Context, commandType int, commandBody string) (string, error) {
	id := uuid.New().String()

	values := map[string]string{"CommandId": id, "CommandType": strconv.Itoa(commandType), "CommandBody": commandBody, "NodeId": n.nodeId}

	jsonValue, _ := json.Marshal(values)

	//TODO: Refactor code to pass struct containing http status code, or error
	ch := make(chan string, 2)

	n.commandChannel[id] = ch

	log.Printf("Channel created: %s on %s", id, n.address)

	defer delete (n.commandChannel, id)
	defer close (ch)

	res, err := common.HttpPostWithContext(context,n.torUrl,bytes.NewBuffer(jsonValue))
	//_, err := http.Post(n.torUrl, "application/json", bytes.NewBuffer(jsonValue))

	if err != nil {
		return "", err
	}
	_ = res
	// Wait
	responseBody := <- ch

	//TODO: should pass correct error instead of nil
	return responseBody, nil
}

func (n NodeProxy) ProcessResponse(commandId string, responseBody string) {
	log.Printf("Channel triggered: %s on %s", commandId, n.address)
	n.commandChannel[commandId] <- responseBody
}

func (n NodeProxy) CreateTransaction(context context.Context, totalIn common.TransactionAmount, fee common.TransactionAmount, totalOut common.TransactionAmount, sourceAddress string) (common.PaymentTransactionReplacing, error) {

	ctx, span := n.tracer.Start(context,"proxy-CreateTransaction-" + n.address)
	defer span.End()

	var request = &models.CreateTransactionCommand{
		TotalIn:       totalIn,
		TotalOut:      totalOut,
		SourceAddress: sourceAddress,
	}

	body, err := json.Marshal(request)

	if err != nil {
		return common.PaymentTransactionReplacing{}, err
	}

	reply, err := n.ProcessCommand(ctx, 0, string(body))

	if err != nil {
		return common.PaymentTransactionReplacing{}, err
	}

	response := &models.CreateTransactionResponse{}

	err = json.Unmarshal([]byte(reply), response)

	if err != nil {
		return common.PaymentTransactionReplacing{}, err
	}

	return response.Transaction, nil
}

func (n NodeProxy) SignTerminalTransactions(context context.Context, creditTransactionPayload *common.PaymentTransactionReplacing) error {

	ctx, span := n.tracer.Start(context,"proxy-SignTerminalTransactions-" + n.address)
	defer span.End()

	traceContext,err :=common.CreateTraceContext(span.SpanContext())

	if err != nil {
		return err
	}

	var request = &models.SignTerminalTransactionCommand{
		Transaction: *creditTransactionPayload,
		Context:	 traceContext,
	}

	body, err := json.Marshal(request)

	if err != nil {
		return err
	}

	reply, err := n.ProcessCommand(ctx, 1,  string(body))

	if err != nil {
		return err
	}

	var response = &models.SignTerminalTransactionResponse{}

	err = json.Unmarshal([]byte(reply), response)

	if err != nil {
		return err
	}

	*creditTransactionPayload = response.Transaction

	return nil
}

func (n NodeProxy) SignChainTransactions(context context.Context, creditTransactionPayload *common.PaymentTransactionReplacing, debitTransactionPayload *common.PaymentTransactionReplacing) error {

	ctx, span := n.tracer.Start(context,"proxy-SignChainTransactions-" + n.address)
	defer span.End()

	traceContext,err :=common.CreateTraceContext(span.SpanContext())

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

	reply, err := n.ProcessCommand(ctx, 2,  string(body))

	if err != nil {
		return err
	}

	var response = &models.SignChainTransactionsResponse{}

	err = json.Unmarshal([]byte(reply), response)

	if err != nil {
		return err
	}

	testutils.Print(&response.Credit.PendingTransaction)
	testutils.Print(&response.Debit.PendingTransaction)

	*creditTransactionPayload = response.Credit
	*debitTransactionPayload = response.Debit

	return nil
}

func (n NodeProxy) CommitServiceTransaction(context context.Context, transaction *common.PaymentTransactionReplacing, pr common.PaymentRequest) (bool, error) {

	ctx, span := n.tracer.Start(context,"proxy-CommitServiceTransaction-" + n.address)
	defer span.End()

	traceContext,err :=common.CreateTraceContext(span.SpanContext())

	if err != nil {
		return false, err
	}

	var request = &models.CommitServiceTransactionCommand {
		Transaction: 	*transaction,
		PaymentRequest: pr,
		Context:		traceContext,
	}

	body, err := json.Marshal(request)

	if err != nil {
		return false, errors.Errorf(err.Error())
	}

	reply, err := n.ProcessCommand(ctx,4, string(body))

	if err != nil {
		return false, errors.Errorf(err.Error())
	}

	var response = &models.CommitServiceTransactionResponse{}

	err = json.Unmarshal([]byte(reply), response)

	if err != nil {
		return false, errors.Errorf(err.Error())
	}

	return response.Ok, nil
}

func (n NodeProxy) CommitPaymentTransaction(context context.Context, transactionPayload *common.PaymentTransactionReplacing) (ok bool, err error) {

	ctx, span := n.tracer.Start(context,"proxy-CommitPaymentTransaction-" + n.address)
	defer span.End()

	traceContext,err :=common.CreateTraceContext(span.SpanContext())

	if err != nil {
		return false, err
	}


	var request = &models.CommitPaymentTransactionCommand {
		Transaction: *transactionPayload,
		Context:	 traceContext,
	}

	body, err := json.Marshal(request)

	if err != nil {
		return false, errors.Errorf(err.Error())
	}

	reply, err := n.ProcessCommand(ctx,3, string(body))

	if err != nil {
		return false, errors.Errorf(err.Error())
	}

	var response = &models.CommitPaymentTransactionResponse{}

	err = json.Unmarshal([]byte(reply), response)

	if err != nil {
		return false, errors.Errorf(err.Error())
	}

	return response.Ok, nil
}



