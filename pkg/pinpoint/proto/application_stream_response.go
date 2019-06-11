package proto

import (
	"encoding/binary"
	"io"
	"net"

	"github.com/imdevlab/g"
	"go.uber.org/zap"
)

// ApplicationStreamResponse ...
type ApplicationStreamResponse struct {
	Type      int16
	ChannelID int
	Length    int
	Payload   []byte
}

// NewApplicationStreamResponse ...
func NewApplicationStreamResponse() *ApplicationStreamResponse {
	return &ApplicationStreamResponse{
		Type: APPLICATION_STREAM_RESPONSE,
	}
}

// Decode ...
func (a *ApplicationStreamResponse) Decode(conn net.Conn, reader io.Reader) error {
	buf := make([]byte, 8)
	if _, err := io.ReadFull(reader, buf); err != nil {
		g.L.Warn("ApplicationResponse Decode", zap.String("error", err.Error()))
		return err
	}
	a.ChannelID = int(binary.BigEndian.Uint32(buf[:4]))
	a.Length = int(binary.BigEndian.Uint32(buf[4:8]))

	a.Payload = make([]byte, a.Length)
	if _, err := io.ReadFull(reader, a.Payload); err != nil {
		return err
	}
	//log.Println(string(a.Payload))
	return nil
}

// Encode ...
func (a *ApplicationStreamResponse) Encode() ([]byte, error) {
	return nil, nil
}

// GetPacketType ...
func (a *ApplicationStreamResponse) GetPacketType() int16 {
	return a.Type
}

// GetPayload ...
func (a *ApplicationStreamResponse) GetPayload() []byte {
	return nil
}

// GetRequestID ...
func (a *ApplicationStreamResponse) GetRequestID() int {
	return 0
}
