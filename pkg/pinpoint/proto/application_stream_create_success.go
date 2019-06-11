package proto

import (
	"encoding/binary"
	"io"
	"net"

	"github.com/imdevlab/g"
	"go.uber.org/zap"
)

// ApplicationStreamCreateSuccess ...
type ApplicationStreamCreateSuccess struct {
	Type      int16
	ChannelID int
}

// NewApplicationStreamCreateSuccess ...
func NewApplicationStreamCreateSuccess() *ApplicationStreamCreateSuccess {
	return &ApplicationStreamCreateSuccess{
		Type: APPLICATION_STREAM_CREATE_SUCCESS,
	}
}

// Decode ...
func (a *ApplicationStreamCreateSuccess) Decode(conn net.Conn, reader io.Reader) error {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(reader, buf); err != nil {
		g.L.Warn("ApplicationRequest Decode", zap.String("error", err.Error()))
		return err
	}
	a.ChannelID = int(binary.BigEndian.Uint32(buf[:4]))
	return nil
}

// Encode ...
func (a *ApplicationStreamCreateSuccess) Encode() ([]byte, error) {
	return nil, nil
}

// GetPacketType ...
func (a *ApplicationStreamCreateSuccess) GetPacketType() int16 {
	return a.Type
}

// GetPayload ...
func (a *ApplicationStreamCreateSuccess) GetPayload() []byte {
	return nil
}

// GetRequestID ...
func (a *ApplicationStreamCreateSuccess) GetRequestID() int {
	return 0
}
