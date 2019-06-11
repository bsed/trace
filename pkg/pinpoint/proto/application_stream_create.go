package proto

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"

	"github.com/imdevlab/g"
	"go.uber.org/zap"
)

// ApplicationStreamCreate ...
type ApplicationStreamCreate struct {
	Type      int16
	ChannelID int
	Length    int
	Payload   []byte
}

// NewApplicationStreamCreate ...
func NewApplicationStreamCreate() *ApplicationStreamCreate {
	return &ApplicationStreamCreate{
		Type: APPLICATION_STREAM_CREATE,
	}
}

// Decode ...
func (a *ApplicationStreamCreate) Decode(conn net.Conn, reader io.Reader) error {
	buf := make([]byte, 8)
	if _, err := io.ReadFull(reader, buf); err != nil {
		g.L.Warn("ApplicationRequest Decode", zap.String("error", err.Error()))
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
func (a *ApplicationStreamCreate) Encode() ([]byte, error) {
	body := make([]byte, 10)
	binary.BigEndian.PutUint16(body[0:2], uint16(a.Type))
	binary.BigEndian.PutUint32(body[2:6], uint32(a.ChannelID))
	binary.BigEndian.PutUint32(body[6:10], uint32(len(a.Payload)))
	bys := bytes.NewBuffer(body)
	bys.Write(a.Payload)
	return bys.Bytes(), nil
}

// GetPacketType ...
func (a *ApplicationStreamCreate) GetPacketType() int16 {
	return a.Type
}

// GetPayload ...
func (a *ApplicationStreamCreate) GetPayload() []byte {
	return nil
}

// GetRequestID ...
func (a *ApplicationStreamCreate) GetRequestID() int {
	return 0
}
