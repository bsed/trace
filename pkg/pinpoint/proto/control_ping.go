package proto

import (
	"encoding/binary"
	"io"
	"net"

	"github.com/imdevlab/g"
	"go.uber.org/zap"
)

// CONTROL_PING
// ControlPing ...
type ControlPing struct {
	Type         int16
	PingID       int
	StateVersion byte
	stateCode    byte
}

func NewControlPing() *ControlPing {
	return &ControlPing{
		Type: CONTROL_PING,
	}
}

func (c *ControlPing) Decode(conn net.Conn, reader io.Reader) error {
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
func (c *ControlPing) Encode() ([]byte, error) {
	return nil, nil
}

// GetPacketType ...
func (c *ControlPing) GetPacketType() int16 {
	return CONTROL_PING
}

// GetPayload ...
func (c *ControlPing) GetPayload() []byte {
	return nil
}

// GetRequestID ...
func (c *ControlPing) GetRequestID() int {
	return 0
}
