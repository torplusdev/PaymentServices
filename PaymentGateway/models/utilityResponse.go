package models

type CommandResponseCore struct {
	SessionId string `json:"sessionId"`
	CommandId string `json:"commandId"`
	NodeId    string `json:"nodeId"`
}
type UtilityResponse struct {
	CommandResponseCore
	CommandResponse []byte `json:"responseBody"`
}

/*
type ProcessCommandResponse struct {
	CommandResponseCore
	Response OutCommandType `json:"responseBody"`
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

func UnmarshalCommandResponseJSON(commandType CommandType, data []byte) (*ProcessCommandResponse, error) {
	typ := &UtilityResponse{}
	if err := json.Unmarshal(data, &typ); err != nil {
		return nil, err
	}
	val, err := CommandType_CommandResponse(commandType)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(typ.CommandResponse, val)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &ProcessCommandResponse{
		CommandResponseCore: typ.CommandResponseCore,
		Response:            val,
	}, nil
}
*/

// func (d *ProcessCommandResponse) UnmarshalJSON(data []byte) error {
// 	var typ struct {
// 		CommandType CommandType `json:"commandType"`
// 		CommandBody []byte      `json:"commandBody"`
// 	}
// 	if err := json.Unmarshal(data, &typ); err != nil {
// 		return err
// 	}
// 	val, err := CommandType_Command(typ.CommandType)
// 	if err != nil {
// 		return err
// 	}
// 	err = json.Unmarshal(typ.CommandBody, val)
// 	if err != nil {
// 		return err
// 	}
// 	if err != nil {
// 		return err
// 	} else {
// 		d.CommandBody = val
// 	}
// 	return nil
// }
