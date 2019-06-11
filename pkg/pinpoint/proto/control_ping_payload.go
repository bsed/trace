package proto

import (
	"encoding/binary"
	"io"
	"net"

	"github.com/imdevlab/g"
	"go.uber.org/zap"
)

// CONTROL_PING_PAYLOAD
// ControlPingPayload ...
type ControlPingPayload struct {
	Type         int16
	PingID       int
	StateVersion byte
	stateCode    byte
}

func NewControlPingPayload() *ControlPingPayload {
	return &ControlPingPayload{
		Type: CONTROL_PING_PAYLOAD,
	}
}

func (c *ControlPingPayload) Decode(conn net.Conn, reader io.Reader) error {
	buf := make([]byte, 6)
	if _, err := io.ReadFull(reader, buf); err != nil {
		g.L.Warn("ControlPing Decode", zap.String("error", err.Error()))
		return err
	}

	c.PingID = int(binary.BigEndian.Uint32(buf[:4]))
	c.StateVersion = buf[4]
	c.stateCode = buf[5]

	return nil
}

// Encode ...
func (c *ControlPingPayload) Encode() ([]byte, error) {
	return nil, nil
}

// GetPacketType ...
func (c *ControlPingPayload) GetPacketType() int16 {
	return CONTROL_PING
}

// GetPayload ...
func (c *ControlPingPayload) GetPayload() []byte {
	return nil
}

// GetRequestID ...
func (c *ControlPingPayload) GetRequestID() int {
	return 0
}
