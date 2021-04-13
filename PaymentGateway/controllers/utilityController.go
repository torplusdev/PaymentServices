package controllers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/correlation"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/plugin/httptrace"
	"paidpiper.com/payment-gateway/commodity"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node"
)

type UtilityController struct {
	node               node.PPNode
	transactionManager node.PPTransactionManager
	requestProvider    node.PPPaymentRequestProvider
	commodityManager   *commodity.Manager
}

func NewUtilityController(node node.PPNode, tm node.PPTransactionManager, rp node.PPPaymentRequestProvider, commodityManager *commodity.Manager) *UtilityController {
	return &UtilityController{
		node:               node,
		commodityManager:   commodityManager,
		transactionManager: tm,
		requestProvider:    rp,
	}
}

func spanFromContext(rootContext context.Context, traceContext common.TraceContext, spanName string) (context.Context, trace.Span) {

	tracer := common.CreateTracer("paidpiper/controller")

	var traceId [16]byte
	var spanId [8]byte

	ba, _ := base64.StdEncoding.DecodeString(traceContext.TraceID)
	copy(traceId[:], ba)

	ba, _ = base64.StdEncoding.DecodeString(traceContext.SpanID)
	copy(spanId[:], ba)

	spanContext := core.SpanContext{
		TraceID:    traceId,
		SpanID:     spanId,
		TraceFlags: traceContext.TraceFlags,
	}

	var span trace.Span
	var ctx context.Context

	if (core.SpanContext{}) == spanContext {
		ctx, span = tracer.Start(rootContext,
			spanName,
		)
	} else {
		ctx, span = tracer.Start(
			trace.ContextWithRemoteSpanContext(rootContext, spanContext),
			spanName,
		)
	}

	return ctx, span
}

func (u *UtilityController) CreateTransaction(context context.Context, commandBody []byte) (interface{}, error) {
	request := &models.CreateTransactionCommand{}

	err := json.Unmarshal(commandBody, request)

	if err != nil {
		return nil, err
	}

	transaction, err := u.node.CreateTransaction(context, request.TotalIn, request.TotalIn-request.TotalOut, request.TotalOut, request.SourceAddress, request.ServiceSessionId)

	if err != nil {
		return nil, err
	}

	response := &models.CreateTransactionResponse{
		Transaction: transaction,
	}

	return response, nil
}

func (u *UtilityController) SignTerminalTransaction(context context.Context, commandBody []byte) (interface{}, error) {
	request := &models.SignTerminalTransactionCommand{}

	err := json.Unmarshal(commandBody, request)

	if err != nil {
		return nil, err
	}

	err = u.node.SignTerminalTransactions(context, &request.Transaction)

	if err != nil {
		return nil, err
	}

	response := models.SignTerminalTransactionResponse{
		Transaction: request.Transaction,
	}

	return response, nil
}

func (u *UtilityController) SignChainTransactions(context context.Context, commandBody []byte) (interface{}, error) {
	request := &models.SignChainTransactionsCommand{}

	err := json.Unmarshal(commandBody, request)

	if err != nil {
		return nil, err
	}

	err = u.node.SignChainTransactions(context, &request.Credit, &request.Debit)

	if err != nil {
		return nil, err
	}

	response := &models.SignChainTransactionsResponse{
		Debit:  request.Debit,
		Credit: request.Credit,
	}

	return response, nil
}

func (u *UtilityController) CommitServiceTransaction(context context.Context, commandBody []byte) (interface{}, error) {

	request := &models.CommitServiceTransactionCommand{}

	err := json.Unmarshal(commandBody, request)

	if err != nil {
		return nil, err
	}

	ctx, span := spanFromContext(context, request.Context, "utility-CommitServiceTransaction")
	defer span.End()

	err = u.node.CommitServiceTransaction(ctx, &request.Transaction, request.PaymentRequest)

	if err != nil {
		return nil, err
	}

	response := &models.CommitServiceTransactionResponse{}

	return response, nil
}

func (u *UtilityController) CommitPaymentTransaction(context context.Context, commandBody []byte) (interface{}, error) {
	request := &models.CommitPaymentTransactionCommand{}

	err := json.Unmarshal(commandBody, request)

	if err != nil {
		return nil, err
	}

	err = u.node.CommitPaymentTransaction(context, &request.Transaction)

	if err != nil {
		return nil, err
	}

	response := &models.CommitPaymentTransactionResponse{}

	return response, nil
}

func spanFromRequest(r *http.Request, spanName string) (context.Context, trace.Span) {

	tracer := common.CreateTracer("paidpiper/controller")
	attrs, entries, spanCtx := httptrace.Extract(r.Context(), r)

	r = r.WithContext(correlation.ContextWithMap(r.Context(), correlation.NewMap(correlation.MapUpdate{
		MultiKV: entries,
	})))

	ctx, span := tracer.Start(
		trace.ContextWithRemoteSpanContext(r.Context(), spanCtx),
		spanName,
		trace.WithAttributes(attrs...),
	)

	return ctx, span
}

func (u *UtilityController) ValidatePayment(w http.ResponseWriter, r *http.Request) {

	_, span := spanFromRequest(r, "ValidatePayment")
	defer span.End()

	request := &models.ValidatePaymentRequest{}

	err := json.NewDecoder(r.Body).Decode(request)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Bad request"))
		return
	}

	paymentRequest := &common.PaymentRequest{}

	err = json.Unmarshal([]byte(request.PaymentRequest), paymentRequest)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Unknown payment request"))
		return
	}

	quantity, err := u.commodityManager.ReverseCalculate(request.ServiceType, request.CommodityType, paymentRequest.Amount, paymentRequest.Asset)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusBadRequest, err.Error()))
		return
	}

	response := &models.ValidatePaymentResponse{
		Quantity: quantity,
	}

	Respond(w, response)
}

func (u *UtilityController) CreatePaymentInfo(w http.ResponseWriter, r *http.Request) {
	ctx, span := spanFromRequest(r, "requesthandler:CreatePaymentInfo")

	defer span.End()

	request := &models.CreatePaymentInfo{}
	err := json.NewDecoder(r.Body).Decode(request)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Invalid request"))
		return
	}

	price, asset, err := u.commodityManager.Calculate(request.ServiceType, request.CommodityType, request.Amount)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Invalid commodity"))
		return
	}

	pr, err := u.requestProvider.CreatePaymentRequest(ctx, price, asset, request.ServiceType)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Invalid request"))
		return
	}

	Respond(w, pr)
}

func (u *UtilityController) ListTransactions(w http.ResponseWriter, r *http.Request) {
	_, span := spanFromRequest(r, "requesthandler:ListTransactions")
	defer span.End()

	trx := u.transactionManager.GetTransactions()

	Respond(w, trx)
}

func (u *UtilityController) GetTransaction(w http.ResponseWriter, r *http.Request) {
	_, span := spanFromRequest(r, "requesthandler:GetTransaction")
	defer span.End()

	vars := mux.Vars(r)
	sessionId := vars["sessionId"]

	trx := u.transactionManager.GetTransaction(sessionId)

	Respond(w, trx)
}

func (u *UtilityController) FlushTransactions(w http.ResponseWriter, r *http.Request) {

	ctx, span := spanFromRequest(r, "requesthandler:FlushTransactions")
	defer span.End()

	results, err := u.transactionManager.FlushTransactions(ctx)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusBadRequest, "Error in FlushTransactions: "+err.Error()))
	}

	for k, v := range results {
		switch v.(type) {
		case error:
			log.Printf("Error in transaction for node %s: %v", k, v)
		default:
		}
	}

	Respond(w, MessageWithStatus(http.StatusOK, "Transactions committed"))
}

func (u *UtilityController) GetStellarAddress(w http.ResponseWriter, r *http.Request) {
	response := &models.GetStellarAddressResponse{
		Address: u.node.GetAddress(),
	}

	Respond(w, response)
}

func (u *UtilityController) GetBalance(w http.ResponseWriter, r *http.Request) {

	response := &models.GetBalanceResponse{
		Balance:   100,
		Timestamp: time.Now(),
	}

	Respond(w, response)
}

func (u *UtilityController) GetUsageStatistics(w http.ResponseWriter, r *http.Request) {

	response := []struct {
		Date  time.Time
		Value int
	}{
		{time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC), 12},
		{time.Date(2021, 3, 2, 0, 0, 0, 0, time.UTC), 32},
		{time.Date(2021, 3, 3, 0, 0, 0, 0, time.UTC), 52},
		{time.Date(2021, 3, 4, 0, 0, 0, 0, time.UTC), 55},
		{time.Date(2021, 3, 5, 0, 0, 0, 0, time.UTC), 57},
		{time.Date(2021, 3, 6, 0, 0, 0, 0, time.UTC), 66},
		{time.Date(2021, 3, 7, 0, 0, 0, 0, time.UTC), 50},
		{time.Date(2021, 3, 8, 0, 0, 0, 0, time.UTC), 80},
		{time.Date(2021, 3, 9, 0, 0, 0, 0, time.UTC), 78},
		{time.Date(2021, 3, 10, 0, 0, 0, 0, time.UTC), 11},
		{time.Date(2021, 3, 11, 0, 0, 0, 0, time.UTC), 38},
		{time.Date(2021, 3, 12, 0, 0, 0, 0, time.UTC), 47},
		{time.Date(2021, 3, 13, 0, 0, 0, 0, time.UTC), 40},
		{time.Date(2021, 3, 14, 0, 0, 0, 0, time.UTC), 86},
		{time.Date(2021, 3, 15, 0, 0, 0, 0, time.UTC), 32},
		{time.Date(2021, 3, 16, 0, 0, 0, 0, time.UTC), 22},
		{time.Date(2021, 3, 17, 0, 0, 0, 0, time.UTC), 48},
		{time.Date(2021, 3, 18, 0, 0, 0, 0, time.UTC), 30},
		{time.Date(2021, 3, 19, 0, 0, 0, 0, time.UTC), 79},
		{time.Date(2021, 3, 20, 0, 0, 0, 0, time.UTC), 59},
		{time.Date(2021, 3, 21, 0, 0, 0, 0, time.UTC), 29},
		{time.Date(2021, 3, 22, 0, 0, 0, 0, time.UTC), 32},
		{time.Date(2021, 3, 23, 0, 0, 0, 0, time.UTC), 68},
		{time.Date(2021, 3, 24, 0, 0, 0, 0, time.UTC), 61},
		{time.Date(2021, 3, 25, 0, 0, 0, 0, time.UTC), 89},
	}
	Respond(w, response)
}

func (u *UtilityController) ProcessCommand(w http.ResponseWriter, r *http.Request) {
	ctx, span := spanFromRequest(r, "requesthandler:ProcessCommand")
	defer span.End()

	command := &models.UtilityCommand{}
	err := json.NewDecoder(r.Body).Decode(command)

	if err != nil {
		log.Fatal(err)

		Respond(w, MessageWithStatus(http.StatusBadRequest, "Invalid request"))
		return
	}

	future := make(chan ResponseMessage)

	handler := func(cmd *models.UtilityCommand, responseChannel chan<- ResponseMessage) {
		asyncMode := false
		callbackUrl := ""

		if cmd.CallbackUrl != "" {
			asyncMode = true
			callbackUrl = cmd.CallbackUrl
		}

		if asyncMode {
			future <- MessageWithStatus(http.StatusCreated, "command submitted")
		}

		var reply interface{}

		switch cmd.CommandType {
		case 0:
			reply, err = u.CreateTransaction(ctx, cmd.CommandBody)
		case 1:
			reply, err = u.SignTerminalTransaction(ctx, cmd.CommandBody)
		case 2:
			reply, err = u.SignChainTransactions(ctx, cmd.CommandBody)
		case 3:
			reply, err = u.CommitPaymentTransaction(ctx, cmd.CommandBody)
		case 4:
			reply, err = u.CommitServiceTransaction(ctx, cmd.CommandBody)
		}

		if asyncMode && err == nil {
			data, err := json.Marshal(reply)

			if err != nil {
				log.Printf("Command response marshal failed: %s", err.Error())
				log.Fatal(err)

				return
			}

			values := &models.ProcessCommandResponse{
				CommandResponse: data,
				CommandId:       cmd.CommandId,
				NodeId:          cmd.NodeId,
				SessionId:       cmd.SessionId,
			}

			jsonValue, _ := json.Marshal(values)

			response, err := common.HttpPostWithoutContext(callbackUrl, bytes.NewBuffer(jsonValue))

			if err != nil {
				log.Printf("Callback url execution failed: : %s", err.Error())
				log.Fatal(err)

				future <- MessageWithStatus(http.StatusConflict, err.Error())

				return
			}

			if response.Body != nil {
				response.Body.Close()
			}

			return
		}

		if err != nil {
			future <- MessageWithStatus(http.StatusConflict, err.Error())
			return
		}

		future <- MessageWithData(http.StatusOK, reply)
	}

	go handler(command, future)

	Respond(w, future)
}
