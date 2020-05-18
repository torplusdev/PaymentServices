package models

type ProcessCommand struct {
	NodeId		string
	CommandId	string
	CommandType int
	CommandBody []byte
}

type ProcessCommandResponse struct {
	CommandResponse	[]byte
	CommandId		string
	NodeId			string
}
