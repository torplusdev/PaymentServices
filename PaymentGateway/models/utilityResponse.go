package models

type UtilityResponse struct {
	CommandId	string 	`json:"commandId"`
	ResponseBody	string	`json:"responseBody"`
	NodeId      string  `json:"nodeId"`
}
