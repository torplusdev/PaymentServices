# Payment process:

Roles:

Client - node which send pptoken and control payment processing. 
Relay1, Relay2, Relay3 - payment proxy nodes.
Service - node which receive pptoken 


Service (decice need payment ) 
Call local pg createPayment 

    /api/utility/createPaymentInfo 
    	{
            ServiceType:   "ipfs",
		CommodityType: "data",
		Amount:        amount,
        }   
    <- 
    {"Amount":1,"Asset":"pptoken","ServiceRef":"tor","ServiceSessionId":"c3hhjnglrabpf17r0qjg","Address":"GARGQG2RJ5UIJRWTFP6E4PYBD2JXAKX2N5DJ3TQACQN2BTUBYFXHIJ4O"}


Clinet: 
/api/utility/validatePayment
{
		PaymentRequest: {
                Amount           TransactionAmount
                Asset            string
                ServiceRef       string
                ServiceSessionId string
                Address          string
            },
		ServiceType:    "ipfs",
		CommodityType:  "data",
	}

/api/gateway/processPayment

/api/command
SessionId:   cl.sessionId,
NodeId:      cl.nodeId,
CommandId:   uuid.New().String(),
CommandType: cmd.Type(),
CommandBody: body,

/api/commandResponse  //to tor
SessionId string `json:"SessionId"`
CommandId string `json:"CommandId"`
NodeId    string `json:"NodeId"`
CommandResponse []byte

TO PG
SessionId string `json:"SessionId"`
CommandId string `json:"CommandId"`
NodeId    string `json:"NodeId"`
ResponseBody    []byte `json:"ResponseBody"`
