package proto

import (
	"encoding/binary"
	"io"
	"net"

	"github.com/imdevlab/g"
	"go.uber.org/zap"
)

// APPLICATION_SEND
// ApplicationSend ...
type ApplicationSend struct {
	Type    int16
	Length  int
	Payload []byte
}

func NewApplicationSend() *ApplicationSend {
	return &ApplicationSend{
		Type: APPLICATION_SEND,
	}
}

// Decode ...
func (a *ApplicationSend) Decode(conn net.Conn, reader io.Reader) error {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(reader, buf); err != nil {
		g.L.Warn("ApplicationSend Decode", zap.String("error", err.Error()))
		return err
	}
	a.Length = int(binary.BigEndian.Uint32(buf[:4]))
	a.Payload = make([]byte, a.Length)

	if _, err := io.ReadFull(reader, a.Payload); err != nil {
		return err
	}
	//log.Println("APPLICATION_SEND", string(a.Payload))
	return nil
}

// Encode ...
func (a *ApplicationSend) Encode() ([]byte, error) {
	return nil, nil
}

// GetPacketType ...
func (a *ApplicationSend) GetPacketType() int16 {
	return APPLICATION_SEND
}

// GetPayload ...
func (a *ApplicationSend) GetPayload() []byte {
	return a.Payload
}

// GetRequestID ...
func (a *ApplicationSend) GetRequestID() int {
	return 0
}
