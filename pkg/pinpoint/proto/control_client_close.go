package proto

import (
	"encoding/binary"
	"io"
	"net"

	"github.com/imdevlab/g"
	"go.uber.org/zap"
)

// ControlClientClose ...
type ControlClientClose struct {
	Type    int16
	Length  int
	Payload []byte
}

// NewControlClientClose ...
func NewControlClientClose() *ControlClientClose {
	return &ControlClientClose{
		Type: CONTROL_CLIENT_CLOSE,
	}
}

// Decode ...
func (c *ControlClientClose) Decode(conn net.Conn, reader io.Reader) error {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(reader, buf); err != nil {
		g.L.Warn("ControlHandShake Decode", zap.String("error", err.Error()))
		return err
	}
	c.Length = int(binary.BigEndian.Uint32(buf[:4]))
	c.Payload = make([]byte, c.Length)

	if _, err := io.ReadFull(reader, c.Payload); err != nil {
		return err
	}

	//log.Println(string(c.Payload))
	return nil
}

// Encode ...
func (c *ControlClientClose) Encode() ([]byte, error) {
	return nil, nil
}

// GetPacketType ...
func (c *ControlClientClose) GetPacketType() int16 {
	return c.Type
}

// GetPayload ...
func (c *ControlClientClose) GetPayload() []byte {
	return nil
}

// GetRequestID ...
func (c *ControlClientClose) GetRequestID() int {
	return 0
}
