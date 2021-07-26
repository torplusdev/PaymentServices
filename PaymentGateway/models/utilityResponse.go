package models

import (
	"encoding/json"
)

type CommandResponseCore struct {
	SessionId   string      `json:"SessionId"`
	CommandId   string      `json:"CommandId"`
	NodeId      string      `json:"NodeId"`
	CommandType CommandType `json:"CommandType"`
}
type UtilityResponse struct { // FROM PG TO TOR or IPFS
	CommandResponseCore
	CommandResponse []byte `json:"CommandResponse"`
}
type UtilityResponseFixModel struct {
	CommandResponseCore
	CommandResponse []byte `json:"CommandResponse"`
}
type ShapelessProcessCommandResponse struct { // FROM TOR TO PG
	CommandResponseCore
	CommandResponse []byte `json:"ResponseBody"`
}

func NewShapelessProcessCommandResponse(im *UtilityResponseFixModel) *ShapelessProcessCommandResponse {
	b := im.CommandResponse
	return &ShapelessProcessCommandResponse{
		CommandResponseCore: im.CommandResponseCore,
		CommandResponse:     b,
	}
}

type ProcessCommandResponse struct {
	CommandResponseCore
	Response OutCommandType `json:"CommandResponse"`
}

func (pr *ProcessCommandResponse) MarshalJSON() ([]byte, error) {
	bs, err := json.Marshal(&pr.Response)
	if err != nil {
		return bs, err
	}

	bs, err = json.Marshal(&UtilityResponse{
		CommandResponseCore: pr.CommandResponseCore,
		CommandResponse:     bs,
	})
	if err != nil {
		return bs, err
	}
	return bs, err
}
