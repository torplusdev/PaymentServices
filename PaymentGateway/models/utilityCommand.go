package models

type UtilityCommand struct {
	CommandId	string 	`json:"commandId"`
	CommandType int		`json:"commandType"`
	CommandBody	string	`json:"commandBody"`
	NodeId      string  `json:"nodeId"`
}