package proto

import (
	"encoding/binary"
	"io"
	"net"

	"github.com/imdevlab/g"
	"go.uber.org/zap"
)

// ApplicationStreamClose ...
type ApplicationStreamClose struct {
	Type      int16
	ChannelID int
	Code      int16
}

// NewApplicationStreamClose ...
func NewApplicationStreamClose() *ApplicationStreamClose {
	return &ApplicationStreamClose{
		Type: APPLICATION_STREAM_CLOSE,
	}
}

// Decode ...
func (a *ApplicationStreamClose) Decode(conn net.Conn, reader io.Reader) error {
	buf := make([]byte, 6)
	if _, err := io.ReadFull(reader, buf); err != nil {
		g.L.Warn("ApplicationRequest Decode", zap.String("error", err.Error()))
		return err
	}

	a.ChannelID = int(binary.BigEndian.Uint32(buf[:4]))
	a.Code = int16(binary.BigEndian.Uint16(buf[4:6]))

	return nil
}

// Encode ...
func (a *ApplicationStreamClose) Encode() ([]byte, error) {
	return nil, nil
}

// GetPacketType ...
func (a *ApplicationStreamClose) GetPacketType() int16 {
	return a.Type
}

// GetPayload ...
func (a *ApplicationStreamClose) GetPayload() []byte {
	return nil
}

// GetRequestID ...
func (a *ApplicationStreamClose) GetRequestID() int {
	return 0
}
