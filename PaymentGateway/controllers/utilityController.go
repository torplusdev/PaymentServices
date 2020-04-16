package controllers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/rs/xid"
	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/correlation"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/plugin/httptrace"
	"log"
	"net/http"
	"paidpiper.com/payment-gateway/common"
	"paidpiper.com/payment-gateway/models"
	"paidpiper.com/payment-gateway/node"
	"strconv"
)

type UtilityController struct {
	Node *node.Node
}

func spanFromContext(rootContext context.Context, traceContext common.TraceContext, spanName string) (context.Context, trace.Span) {

	tracer := global.Tracer("paidpiper/controller")

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

func (u *UtilityController) CreateTransaction(context context.Context, commandBody string) (interface{}, error) {
	request := &models.CreateTransactionCommand{}

	err := json.Unmarshal([]byte(commandBody), request)

	if err != nil {
		return nil, err
	}

	transaction, err := u.Node.CreateTransaction(context, request.TotalIn, request.TotalIn-request.TotalOut, request.TotalOut, request.SourceAddress)

	if err != nil {
		return nil, err
	}

	response := &models.CreateTransactionResponse{
		Transaction: transaction,
	}

	return response, nil
}

func (u *UtilityController) SignTerminalTransaction(context context.Context, commandBody string) (interface{}, error) {
	request := &models.SignTerminalTransactionCommand{}

	err := json.Unmarshal([]byte(commandBody), request)

	if err != nil {
		return nil, err
	}

	err = u.Node.SignTerminalTransactions(context, &request.Transaction)

	if err != nil {
		return nil, err
	}

	response := models.SignTerminalTransactionResponse{
		Transaction: request.Transaction,
	}

	return response, nil
}

func (u *UtilityController) SignChainTransactions(context context.Context, commandBody string) (interface{}, error) {
	request :=  &models.SignChainTransactionsCommand{}

	err := json.Unmarshal([]byte(commandBody), request)

	if err != nil {
		return nil, err
	}

	err = u.Node.SignChainTransactions(context, &request.Credit, &request.Debit)

	if err != nil {
		return nil, err
	}

	response :=  &models.SignChainTransactionsResponse{
		Debit:  request.Debit,
		Credit: request.Credit,
	}

	return response, nil
}

func (u *UtilityController) CommitServiceTransaction(context context.Context, commandBody string) (interface{}, error) {

	request := &models.CommitServiceTransactionCommand{}

	err := json.Unmarshal([]byte(commandBody), request)

	if err != nil {
		return nil, err
	}

	ctx, span := spanFromContext(context,request.Context,"utility-CommitServiceTransaction")
	defer span.End()

	ok, err := u.Node.CommitServiceTransaction(ctx, &request.Transaction, request.PaymentRequest)

	if err != nil {
		return nil, err
	}

	response := &models.CommitServiceTransactionResponse{
		Ok: ok,
	}

	return response, nil
}

func (u *UtilityController) CommitPaymentTransaction(context context.Context, commandBody string) (interface{}, error) {
	request := &models.CommitPaymentTransactionCommand{}

	err := json.Unmarshal([]byte(commandBody), request)

	if err != nil {
		return nil, err
	}

	ok, err := u.Node.CommitPaymentTransaction(context, &request.Transaction)

	if err != nil {
		return nil, err
	}

	response := &models.CommitPaymentTransactionResponse{
		Ok: ok,
	}

	return response, nil
}

func spanFromRequest(r *http.Request, spanName string) (context.Context, trace.Span) {

	tracer := global.Tracer("paidpiper/controller")
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

func (u *UtilityController) CreatePaymentInfo(w http.ResponseWriter, r *http.Request) {

	ctx,span := spanFromRequest(r,"requesthandler:CreatePaymentInfo")

	defer span.End()

	params := mux.Vars(r)

	strAmount := params["amount"]

	serviceSessionId := xid.New().String()

	amount, err := strconv.Atoi(strAmount)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusInternalServerError,"Invalid request"))
		return
	}

	err = u.Node.AddPendingServicePayment(ctx, serviceSessionId, uint32(amount))

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusInternalServerError,"Invalid request"))
		return
	}

	pr, err := u.Node.CreatePaymentRequest(ctx, serviceSessionId)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusInternalServerError,"Invalid request"))
		return
	}

	Respond(w, pr)
}


func (u *UtilityController) FlushTransactions(w http.ResponseWriter, r *http.Request) {

	ctx,span := spanFromRequest(r,"requesthandler:FlushTransactions")
	defer span.End()

	results,err := u.Node.FlushTransactions(ctx)

	if err != nil {
		Respond(w,MessageWithStatus(http.StatusInternalServerError,"Error in FlushTransactions:..."))
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
		Address: u.Node.Address,
	}

	Respond(w, response)
}

func (u *UtilityController) ProcessCommand(w http.ResponseWriter, r *http.Request) {

	ctx,span := spanFromRequest(r,"requesthandler:ProcessCommand")
	defer span.End()

	command := &models.UtilityCommand{}
	err := json.NewDecoder(r.Body).Decode(command)

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusInternalServerError,"Invalid request"))
		return
	}

	var reply interface{}

	switch command.CommandType {
	case 0:
		reply, err = u.CreateTransaction(ctx, command.CommandBody)
	case 1:
		reply, err = u.SignTerminalTransaction(ctx, command.CommandBody)
	case 2:
		reply, err = u.SignChainTransactions(ctx, command.CommandBody)
	case 3:
		reply, err = u.CommitPaymentTransaction(ctx, command.CommandBody)
	case 4:
		reply, err = u.CommitServiceTransaction(ctx, command.CommandBody)
	}

	if err != nil {
		Respond(w, MessageWithStatus(http.StatusInternalServerError,"Request process failed"))
		return
	}

	Respond(w, reply)
}