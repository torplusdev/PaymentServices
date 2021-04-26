package models

import (
	"encoding/json"
	"fmt"
)

type CommandType int32

func (ct CommandType) String() string {
	switch ct {
	case CommandType_CreateTransaction:
		return "CreateTransaction"
	case CommandType_SignServiceTransaction:
		return "SignTerminalTransaction"

	case CommandType_SignChainTransaction:
		return "SignChainTransaction"

	case CommandType_CommitChainTransaction:
		return "CommitChainTransaction"

	case CommandType_CommitServiceTransaction:
		return "CommitServiceTransaction"
	default:
		return "none"
	}
}

const (
	CommandType_CreateTransaction = iota
	CommandType_SignServiceTransaction
	CommandType_SignChainTransaction
	CommandType_CommitChainTransaction
	CommandType_CommitServiceTransaction
)

type InCommandType interface {
	Type() CommandType
}

func CommandType_Command(c CommandType) (InCommandType, error) {
	switch c {
	case CommandType_CreateTransaction:
		return &CreateTransactionCommand{}, nil
	case CommandType_SignServiceTransaction:
		return &CreateTransactionCommand{}, nil
	case CommandType_SignChainTransaction:
		return &CreateTransactionCommand{}, nil
	case CommandType_CommitChainTransaction:
		return &CreateTransactionCommand{}, nil
	case CommandType_CommitServiceTransaction:
		return &CreateTransactionCommand{}, nil
	default:
		return nil, fmt.Errorf("command type not found")
	}
}

type OutCommandType interface {
	OutType() CommandType
}

func CommandType_CommandResponse(c CommandType) (OutCommandType, error) {
	switch c {
	case CommandType_CreateTransaction:
		return &CreateTransactionResponse{}, nil
	case CommandType_SignServiceTransaction:
		return &CreateTransactionResponse{}, nil
	case CommandType_SignChainTransaction:
		return &CreateTransactionResponse{}, nil
	case CommandType_CommitChainTransaction:
		return &CreateTransactionResponse{}, nil
	case CommandType_CommitServiceTransaction:
		return &CreateTransactionResponse{}, nil
	default:
		return nil, fmt.Errorf("command response type not found")
	}
}

type CommandCore struct {
	SessionId   string      `json:"sessionId"`
	NodeId      string      `json:"nodeId"`
	CommandId   string      `json:"commandId"`
	CommandType CommandType `json:"commandType"`
}
type UtilityCommand struct {
	CommandCore
	CommandBody InCommandType `json:"commandBody"` //`json:"-"` //
	CallbackUrl string        `json:"callbackUrl"`
}
type ShapelessUtilityCommand struct {
	CommandCore
	CommandBody []byte `json:"commandBody"`
	CallbackUrl string `json:"callbackUrl"` //TODO TO OTHER LAYER
}
type ProcessCommand struct {
	CommandCore
	CommandBody []byte `json:"commandBody"`
}

func (pr *UtilityCommand) MarshalJSON() ([]byte, error) {
	bs, err := json.Marshal(&pr.CommandBody)
	if err != nil {
		return bs, err
	}

	bs, err = json.Marshal(&ProcessCommand{
		CommandCore: pr.CommandCore,
		CommandBody: bs,
	})
	if err != nil {
		return bs, err
	}
	return bs, err
}

func (d *UtilityCommand) UnmarshalJSON(data []byte) error {
	typ := &ProcessCommand{}
	if err := json.Unmarshal(data, &typ); err != nil {
		return err
	}
	d.CommandCore = typ.CommandCore
	val, err := CommandType_Command(typ.CommandType)
	if err != nil {
		return err
	}
	err = json.Unmarshal(typ.CommandBody, val)
	if err != nil {
		return err
	}
	if err != nil {
		return err
	} else {
		d.CommandBody = val
	}
	return nil
}

type GetBalanceResponse struct {
	Balance   float64
	Timestamp JsonTime
}
