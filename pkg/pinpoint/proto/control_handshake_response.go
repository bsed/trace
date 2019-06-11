package proto

import (
	"encoding/binary"
	"io"
	"net"
)

// ControlHandShake
type ControlHandShakeResponse struct {
	Type      int16
	RequestID int
	Length    int
	Payload   []byte
}

// NewControlHandShake ...
func NewControlHandShakeResponse() *ControlHandShakeResponse {
	return &ControlHandShakeResponse{
		Type: CONTROL_HANDSHAKE_RESPONSE,
	}
}

// Decode ...
func (c *ControlHandShakeResponse) Decode(conn net.Conn, reader io.Reader) error {
	return nil
}

// Encode ...
func (c *ControlHandShakeResponse) Encode() ([]byte, error) {
	body := make([]byte, 10)
	binary.BigEndian.PutUint16(body[0:2], uint16(c.Type))
	binary.BigEndian.PutUint32(body[2:6], uint32(c.RequestID))
	binary.BigEndian.PutUint32(body[6:10], uint32(len(c.Payload)))
	//bys := bytes.NewBuffer(body)
	//bys.Write(c.Payload)
	//return bys.Bytes(), nil
	body = append(body, c.Payload...)
	return body, nil
}

// GetPacketType ...
func (c *ControlHandShakeResponse) GetPacketType() int16 {
	return c.Type
}

// GetPayload ...
func (c *ControlHandShakeResponse) GetPayload() []byte {
	return c.Payload
}

// GetRequestID ...
func (c *ControlHandShakeResponse) GetRequestID() int {
	return c.RequestID
}
