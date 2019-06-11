package channel

import "github.com/bsed/trace/pkg/alert"

// Message app消息推送
type Message struct {
}

// NewMessage ...
func NewMessage() *Message {
	return &Message{}
}

// AlertPush ...
func (m *Message) AlertPush(msg *alert.Alert) error {
	return nil
}
