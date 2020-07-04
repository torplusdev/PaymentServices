package models

type ProcessCommand struct {
	SessionId   string
	NodeId      string
	CommandId   string
	CommandType int
	CommandBody []byte
}

type ProcessCommandResponse struct {
	CommandResponse []byte
	CommandId       string
	NodeId          string
	SessionId       string
}
