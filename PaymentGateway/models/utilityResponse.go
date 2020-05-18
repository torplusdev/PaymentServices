package models

type UtilityResponse struct {
	CommandId		string 	`json:"commandId"`
	ResponseBody	[]byte	`json:"responseBody"`
	NodeId      	string  `json:"nodeId"`
}
