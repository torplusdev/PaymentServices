package models

import "encoding/json"

type CommandResponseCore struct {
	SessionId string `json:"SessionId"`
	CommandId string `json:"CommandId"`
	NodeId    string `json:"NodeId"`
}
type UtilityResponse struct { // FROM PG TO TOR
	CommandResponseCore
	CommandResponse []byte `json:"CommandResponse"`
}
type ShapelessProcessCommandResponse struct { // FROM TOR TO PG
	CommandResponseCore
	CommandResponse []byte `json:"ResponseBody"`
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
