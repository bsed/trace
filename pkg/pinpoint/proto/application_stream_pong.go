package proto

import (
	"encoding/binary"
	"io"
	"net"

	"github.com/imdevlab/g"
	"go.uber.org/zap"
)

// ApplicationStreamPong ...
type ApplicationStreamPong struct {
	Type      int16
	ChannelID int
	RequestID int
}

// NewApplicationStreamPong ...
func NewApplicationStreamPong() *ApplicationStreamPong {
	return &ApplicationStreamPong{
		Type: APPLICATION_STREAM_PONG,
	}
}

// Decode ...
func (a *ApplicationStreamPong) Decode(conn net.Conn, reader io.Reader) error {
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
func (a *ApplicationStreamPong) Encode() ([]byte, error) {
	return nil, nil
}

// GetPacketType ...
func (a *ApplicationStreamPong) GetPacketType() int16 {
	return a.Type
}

// GetPayload ...
func (a *ApplicationStreamPong) GetPayload() []byte {
	return nil
}

// GetRequestID ...
func (a *ApplicationStreamPong) GetRequestID() int {
	return a.RequestID
}
