package proto

import (
	"encoding/binary"
	"io"
	"net"

	"github.com/imdevlab/g"
	"go.uber.org/zap"
)

// APPLICATION_REQUEST
// ApplicationSend ...
type ApplicationRequest struct {
	Type      int16
	RequestID int
	Length    int
	Payload   []byte
}

// NewApplicationRequest ....
func NewApplicationRequest() *ApplicationRequest {
	return &ApplicationRequest{
		Type: APPLICATION_REQUEST,
	}
}

// Decode ...
func (a *ApplicationRequest) Decode(conn net.Conn, reader io.Reader) error {
	buf := make([]byte, 8)
	if _, err := io.ReadFull(reader, buf); err != nil {
		g.L.Warn("ApplicationRequest Decode", zap.String("error", err.Error()))
		return err
	}
	a.RequestID = int(binary.BigEndian.Uint32(buf[0:4]))
	a.Length = int(binary.BigEndian.Uint32(buf[4:8]))
	a.Payload = make([]byte, a.Length)
	if _, err := io.ReadFull(reader, a.Payload); err != nil {
		return err
	}

	return nil
}

// Encode ...
func (a *ApplicationRequest) Encode() ([]byte, error) {
	return nil, nil
}

// GetPacketType ...
func (a *ApplicationRequest) GetPacketType() int16 {
	return APPLICATION_REQUEST
}

// GetPayload ...
func (a *ApplicationRequest) GetPayload() []byte {
	return a.Payload
}

// GetRequestID ...
func (a *ApplicationRequest) GetRequestID() int {
	return a.RequestID
}
