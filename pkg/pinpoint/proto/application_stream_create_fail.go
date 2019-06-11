package proto

import (
	"encoding/binary"
	"io"
	"net"

	"github.com/imdevlab/g"
	"go.uber.org/zap"
)

// ApplicationStreamCreateFail ...
type ApplicationStreamCreateFail struct {
	Type      int16
	ChannelID int
	Code      int16
}

// NewApplicationStreamCreateFail ...
func NewApplicationStreamCreateFail() *ApplicationStreamCreateFail {
	return &ApplicationStreamCreateFail{
		Type: APPLICATION_STREAM_CREATE_FAIL,
	}
}

// Decode ...
func (a *ApplicationStreamCreateFail) Decode(conn net.Conn, reader io.Reader) error {
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
func (a *ApplicationStreamCreateFail) Encode() ([]byte, error) {
	return nil, nil
}

// GetPacketType ...
func (a *ApplicationStreamCreateFail) GetPacketType() int16 {
	return a.Type
}

// GetPayload ...
func (a *ApplicationStreamCreateFail) GetPayload() []byte {
	return nil
}

// GetRequestID ...
func (a *ApplicationStreamCreateFail) GetRequestID() int {
	return 0
}
