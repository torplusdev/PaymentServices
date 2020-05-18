package models

type ProcessCommand struct {
	NodeId		string
	CommandId	string
	CommandType int
	CommandBody []byte
}
