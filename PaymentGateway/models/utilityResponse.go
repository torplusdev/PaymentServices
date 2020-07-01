package models

type UtilityResponse struct {
	SessionId    string `json:"sessionId"`
	CommandId    string `json:"commandId"`
	ResponseBody []byte `json:"responseBody"`
	NodeId       string `json:"nodeId"`
}
