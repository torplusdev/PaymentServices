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
		return &SignServiceTransactionCommand{}, nil
	case CommandType_SignChainTransaction:
		return &SignChainTransactionCommand{}, nil
	case CommandType_CommitChainTransaction:
		return &CommitChainTransactionCommand{}, nil
	case CommandType_CommitServiceTransaction:
		return &CommitServiceTransactionCommand{}, nil
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
	SessionId   string      `json:"SessionId"`
	NodeId      string      `json:"NodeId"`
	CommandId   string      `json:"CommandId"`
	CommandType CommandType `json:"CommandType"`
}
type UtilityCommand struct {
	CommandCore
	CommandBody InCommandType `json:"CommandBody"` //`json:"-"` //
	CallbackUrl string        `json:"CallbackUrl"`
}

func (pr UtilityCommand) MarshalJSON() ([]byte, error) {
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
	typ := &ShapelessUtilityCommand{}
	if err := json.Unmarshal(data, &typ); err != nil {
		return err
	}
	d.CommandCore = typ.CommandCore
	d.CallbackUrl = typ.CallbackUrl
	val, err := CommandType_Command(typ.CommandType)
	if err != nil {
		return err
	}
	commandBody := typ.CommandBody
	if err = json.Unmarshal(commandBody, val); err != nil {
		return fmt.Errorf("unmarshal error: json %v err: %v", string(commandBody), err)
	}
	d.CommandBody = val
	return nil
}

type ShapelessUtilityCommand struct {
	CommandCore
	CommandBody []byte `json:"CommandBody"`
	CallbackUrl string `json:"CallbackUrl"`
}
type ProcessCommand struct {
	CommandCore
	CommandBody []byte `json:"CommandBody"`
}

type GetBalanceResponse struct {
	Balance   float64
	Timestamp JsonTime
}
