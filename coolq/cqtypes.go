package coolq

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

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

// Marshal 序列化一个消息为cq的消息格式
func Marshal(msg Message) []byte {
	buff := new(bytes.Buffer)
	for _, section := range msg {
		switch section.Type {
		case "text":
			buff.WriteString(section.Data["text"])
		default:
			buff.WriteByte('[')
			buff.WriteString("CQ:")
			buff.WriteString(section.Type)
			for key, val := range section.Data {
				buff.WriteString(fmt.Sprintf(",%s=%s", key, val))
			}
			buff.WriteByte(']')
		}
	}
	return buff.Bytes()
}

// Unmarshal 反序列化bytes为一个msg
func Unmarshal(raw []byte, msg *Message) error {
	idx := 0
	cur := 0
	tot := len(raw)
	for idx < tot {
		if raw[idx] != '[' {
			cur = idx
			for raw[cur] != '[' && cur < tot {
				cur++
			}
			section := Section{
				Type: "text",
				Data: map[string]string{},
			}
			section.Data["text"] = string(raw[idx:cur])
			*msg = append(*msg, section)
			idx = cur
		} else {
			cur = idx
			for raw[cur] != ']' && cur < tot {
				cur++
			}
			if cur == tot {
				msg = nil
				return errors.New("syntax error: unexpected EOF, expecting ]")
			}
			cur++
			cqCode := string(raw[idx:cur])
			cqCode = strings.TrimPrefix(cqCode, "[")
			cqCode = strings.TrimSuffix(cqCode, "]")
			payloads := strings.Split(cqCode, ",")
			filedLen := len(payloads)
			if filedLen < 2 {
				msg = nil
				return errors.New("syntax error: invalid cqcode, expecting one field at least")
			}
			cqType := strings.Split(payloads[0], ":")
			if cqType[0] != "CQ" && len(cqType) != 2 {
				msg = nil
				return errors.New("syntax error: invalid cqcode, expecting a string startswith \"CQ\", and cqcode type after")
			}
			section := Section{
				Type: cqType[1],
				Data: map[string]string{},
			}
			for i := 1; i < filedLen; i++ {
				pair := strings.Split(payloads[i], "=")
				if len(pair) != 2 {
					msg = nil
					errMsg := fmt.Sprintf("syntax error: invalid cqcode, wrong argument, at %s", payloads[i])
					return errors.New(errMsg)
				}
				section.Data[pair[0]] = pair[1]
			}
			*msg = append(*msg, section)
			idx = cur
		}
	}
	return nil
}
