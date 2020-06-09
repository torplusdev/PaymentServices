package controllers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/correlation"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/plugin/httptrace"
	"log"
	"net/http"
	"paidpiper.com/payment-gateway/commodity"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node"
)

type UtilityController struct {
	node             		*node.Node
	commodityManager 		*commodity.Manager
}

func NewUtilityController(node *node.Node, commodityManager *commodity.Manager) *UtilityController {
	return &UtilityController{
		node:             		node,
		commodityManager: 		commodityManager,
	}
}

func spanFromContext(rootContext context.Context, traceContext common.TraceContext, spanName string) (context.Context, trace.Span) {

	tracer := common.CreateTracer("paidpiper/controller")

	var traceId [16]byte
	var spanId [8]byte

	ba,_ := base64.StdEncoding.DecodeString(traceContext.TraceID)
	copy(traceId[:],ba)

	ba,_ = base64.StdEncoding.DecodeString(traceContext.SpanID)
	copy(spanId[:],ba)


	spanContext := core.SpanContext{
		TraceID:    traceId,
		SpanID:     spanId,
		TraceFlags: traceContext.TraceFlags,
	}

	var span trace.Span
	var ctx context.Context

	if (core.SpanContext {}) == spanContext {
		ctx, span = tracer.Start(rootContext,
			spanName,
		)
	} else {
		ctx, span = tracer.Start(
			trace.ContextWithRemoteSpanContext(rootContext, spanContext),
			spanName,
		)
	}

	return ctx,span
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
	request :=  &models.SignChainTransactionsCommand{}

	err := json.Unmarshal(commandBody, request)

	if err != nil {
		return nil, err
	}

	err = u.node.SignChainTransactions(context, &request.Credit, &request.Debit)

	if err != nil {
		return nil, err
	}

	response :=  &models.SignChainTransactionsResponse{
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

	ctx, span := spanFromContext(context,request.Context,"utility-CommitServiceTransaction")
	defer span.End()

	ok, err := u.node.CommitServiceTransaction(ctx, &request.Transaction, request.PaymentRequest)

	if err != nil {
		return nil, err
	}

	response := &models.CommitServiceTransactionResponse{
		Ok: ok,
	}

	return response, nil
}

func (u *UtilityController) CommitPaymentTransaction(context context.Context, commandBody []byte) (interface{}, error) {
	request := &models.CommitPaymentTransactionCommand{}

	err := json.Unmarshal(commandBody, request)

	if err != nil {
		return nil, err
	}

	ok, err := u.node.CommitPaymentTransaction(context, &request.Transaction)

	if err != nil {
		return nil, err
	}

	response := &models.CommitPaymentTransactionResponse{
		Ok: ok,
	}

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

	return ctx,span
}


func (u *UtilityController) ValidatePayment(w http.ResponseWriter, r *http.Request) {

	_, span := spanFromRequest(r,"ValidatePayment")
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

	response := &models.ValidatePaymentResponse {
		Quantity: quantity,
	}

	Respond(w, response)
}

func (u *UtilityController) CreatePaymentInfo(w http.ResponseWriter, r *http.Request) {
	ctx,span := spanFromRequest(r,"requesthandler:CreatePaymentInfo")

	defer span.End()

	request := &models.CreatePaymentInfo{}
	err := json.NewDecoder(r.Body).Decode(request)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusBadRequest,"Invalid request"))
		return
	}

	price, asset, err := u.commodityManager.Calculate(request.ServiceType, request.CommodityType, request.Amount)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusBadRequest,"Invalid commodity"))
		return
	}

	pr, err := u.node.CreatePaymentRequest(ctx, price, asset, request.ServiceType)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusBadRequest,"Invalid request"))
		return
	}

	Respond(w, pr)
}

func (u *UtilityController) ListTransactions(w http.ResponseWriter, r *http.Request) {
	_, span := spanFromRequest(r,"requesthandler:ListTransactions")
	defer span.End()

	trx := u.node.GetTransactions()

	Respond(w, trx)
}

func (u *UtilityController) FlushTransactions(w http.ResponseWriter, r *http.Request) {

	ctx,span := spanFromRequest(r,"requesthandler:FlushTransactions")
	defer span.End()

	results,err := u.node.FlushTransactions(ctx)

	if err != nil {
		Respond(w,MessageWithStatus(http.StatusBadRequest,"Error in FlushTransactions:..."))
	}

	for k,v := range results {
		switch v.(type) {
			case error:
				log.Printf("Error in transaction for node %s: %w",k,v)
			default:
		}
	}

	Respond(w, MessageWithStatus(http.StatusOK,"Transactions committed"))
}

func (u *UtilityController) GetStellarAddress(w http.ResponseWriter, r *http.Request) {
	response := &models.GetStellarAddressResponse {
		Address: u.node.Address,
	}

	Respond(w, response)
}

func (u *UtilityController) ProcessCommand(w http.ResponseWriter, r *http.Request) {

	ctx, span := spanFromRequest(r, "requesthandler:ProcessCommand")
	defer span.End()

	command := &models.UtilityCommand{}
	err := json.NewDecoder(r.Body).Decode(command)

	if err != nil {
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

				return
			}

			values := &models.ProcessCommandResponse{
				CommandResponse: 	data,
				CommandId:    		cmd.CommandId,
				NodeId:       		cmd.NodeId,
				SessionId:			cmd.SessionId,
			}

			jsonValue, _ := json.Marshal(values)

			common.HttpPostWithoutContext(callbackUrl, bytes.NewBuffer(jsonValue))

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