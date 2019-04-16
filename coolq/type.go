package coolq

type websocketSystem struct {
	MsgType string `json:"msg_type"`
	Data    string `json:"data"`
}

// ResultType msg result type
type ResultType string

// ResultType
const (
	SUCCESS ResultType = "SUCCESS"
	FAIL               = "FAIL"
	ERROR              = "ERROR"
)

// ProtoType msg type
type ProtoType string

// ProtoType
const (
	SYSTEM     ProtoType = "SYSTEM"
	NON_SYSTEM           = "NON_SYSTEM"
)
