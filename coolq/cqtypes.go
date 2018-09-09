package coolq

// Number number
type Number int64

// String string
type String string

// Section 消息段落
// https://cqhttp.cc/docs/4.4/#/Message
type Section struct {
	Type string            `json:"type"`
	Data map[string]string `json:"data"`
}

// Message 酷q消息
type Message []Section
