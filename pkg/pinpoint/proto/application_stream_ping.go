package proto

import (
	"encoding/binary"
	"io"
	"net"

	"github.com/imdevlab/g"
	"go.uber.org/zap"
)

// ApplicationStreamPing ...
type ApplicationStreamPing struct {
	Type      int16
	ChannelID int
	RequestID int
}

// NewApplicationStreamPing ...
func NewApplicationStreamPing() *ApplicationStreamPing {
	return &ApplicationStreamPing{
		Type: APPLICATION_STREAM_PING,
	}
}

// Decode ...
func (a *ApplicationStreamPing) Decode(conn net.Conn, reader io.Reader) error {
	buf := make([]byte, 8)
	if _, err := io.ReadFull(reader, buf); err != nil {
		g.L.Warn("ApplicationRequest Decode", zap.String("error", err.Error()))
		return err
	}
	a.ChannelID = int(binary.BigEndian.Uint32(buf[:4]))
	a.RequestID = int(binary.BigEndian.Uint32(buf[4:8]))

	return nil
}

// Encode ...
func (a *ApplicationStreamPing) Encode() ([]byte, error) {
	return nil, nil
}

// GetPacketType ...
func (a *ApplicationStreamPing) GetPacketType() int16 {
	return a.Type
}

// GetPayload ...
func (a *ApplicationStreamPing) GetPayload() []byte {
	return nil
}

// GetRequestID ...
func (a *ApplicationStreamPing) GetRequestID() int {
	return a.RequestID
}
