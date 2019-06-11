package proto

import (
	"encoding/binary"
	"io"
	"net"

	"github.com/imdevlab/g"
	"go.uber.org/zap"
)

// ControlHandShake
type ControlHandShake struct {
	Type      int16
	RequestID int
	Length    int
	Payload   []byte
}

// NewControlHandShake ...
func NewControlHandShake() *ControlHandShake {
	return &ControlHandShake{
		Type: CONTROL_HANDSHAKE,
	}
}

// Decode ...
func (c *ControlHandShake) Decode(conn net.Conn, reader io.Reader) error {
	buf := make([]byte, 8)
	if _, err := io.ReadFull(reader, buf); err != nil {
		g.L.Warn("ControlHandShake Decode", zap.String("error", err.Error()))
		return err
	}
	c.RequestID = int(binary.BigEndian.Uint32(buf[:4]))
	c.Length = int(binary.BigEndian.Uint32(buf[4:8]))
	c.Payload = make([]byte, c.Length)

	if _, err := io.ReadFull(reader, c.Payload); err != nil {
		return err
	}

	//log.Println(string(c.Payload))
	return nil
}

// Encode ...
func (c *ControlHandShake) Encode() ([]byte, error) {
	return nil, nil
}

// GetPacketType ...
func (c *ControlHandShake) GetPacketType() int16 {
	return c.Type
}

// GetPayload ...
func (c *ControlHandShake) GetPayload() []byte {
	return c.Payload
}

// GetRequestID ...
func (c *ControlHandShake) GetRequestID() int {
	return c.RequestID
}
