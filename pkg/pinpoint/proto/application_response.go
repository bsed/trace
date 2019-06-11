package proto

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"

	"github.com/imdevlab/g"
	"go.uber.org/zap"
)

// ApplicationResponse ...
type ApplicationResponse struct {
	Type      int16
	RequestID int
	Length    int
	Payload   []byte
}

func NewApplicationResponse() *ApplicationResponse {
	return &ApplicationResponse{
		Type: APPLICATION_RESPONSE,
	}
}

// Decode ...
func (a *ApplicationResponse) Decode(conn net.Conn, reader io.Reader) error {
	buf := make([]byte, 8)
	if _, err := io.ReadFull(reader, buf); err != nil {
		g.L.Warn("ApplicationResponse Decode", zap.String("error", err.Error()))
		return err
	}
	a.RequestID = int(binary.BigEndian.Uint32(buf[:4]))
	a.Length = int(binary.BigEndian.Uint32(buf[4:8]))

	a.Payload = make([]byte, a.Length)
	if _, err := io.ReadFull(reader, a.Payload); err != nil {
		return err
	}
	//log.Println(string(a.Payload))
	return nil
}

// Encode ...
func (a *ApplicationResponse) Encode() ([]byte, error) {
	body := make([]byte, 10)
	binary.BigEndian.PutUint16(body[0:2], uint16(a.Type))
	binary.BigEndian.PutUint32(body[2:6], uint32(a.RequestID))
	binary.BigEndian.PutUint32(body[6:10], uint32(len(a.Payload)))
	bys := bytes.NewBuffer(body)
	bys.Write(a.Payload)
	return bys.Bytes(), nil
	//body = append(body, a.Payload...)
	return body, nil
}

// GetPacketType ...
func (a *ApplicationResponse) GetPacketType() int16 {
	return APPLICATION_RESPONSE
}

// GetPayload ...
func (a *ApplicationResponse) GetPayload() []byte {
	return a.Payload
}

// GetRequestID ...
func (a *ApplicationResponse) GetRequestID() int {
	return a.RequestID
}
