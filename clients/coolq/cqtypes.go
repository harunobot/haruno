package coolq

// Number number
type Number int64

// String string
type String string

// Section 消息段落
// https://cqhttp.cc/docs/4.3/#/Message
type Section struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Message 酷q消息
type Message []Section
