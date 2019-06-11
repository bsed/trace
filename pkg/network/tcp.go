package network

import (
	"encoding/binary"
	"io"

	"github.com/golang/snappy"
	"github.com/imdevlab/g"
	"github.com/bsed/trace/pkg/constant"
	"go.uber.org/zap"
)

// TracePack trace通信标准包
type TracePack struct {
	Type       byte   `msgp:"t"`  // 类型
	IsSync     byte   `msgp:"is"` // 是否同步
	IsCompress byte   `msgp:"cp"` // 是否压缩
	ID         uint32 `msgp:"id"` // 报文ID
	Len        uint32 `msgp:"l"`  // 长度
	Payload    []byte `msgp:"p"`  // 数据
}

// NewTracePack ...
func NewTracePack() *TracePack {
	return &TracePack{}
}

// Encode encode
func (t *TracePack) Encode() []byte {
	// 压缩
	if t.IsCompress == constant.TypeOfCompressYes {
		if len(t.Payload) > 0 {
			compressBuf := snappy.Encode(nil, t.Payload)
			t.Payload = compressBuf
		}
	}

	t.Len = uint32(len(t.Payload))
	buf := make([]byte, t.Len+11)

	buf[0] = t.Type
	buf[1] = t.IsSync
	buf[2] = t.IsCompress
	binary.BigEndian.PutUint32(buf[3:7], t.ID)
	binary.BigEndian.PutUint32(buf[7:11], t.Len)

	if t.Len > 0 {
		copy(buf[11:], t.Payload)
	}
	return buf

}

// Decode decode
func (t *TracePack) Decode(rdr io.Reader) error {
	buf := make([]byte, 11)
	if _, err := io.ReadFull(rdr, buf); err != nil {
		g.L.Warn("Decode:io.ReadFull", zap.String("err", err.Error()))
		return err
	}

	t.Type = buf[0]
	t.IsSync = buf[1]
	t.IsCompress = buf[2]
	t.ID = binary.BigEndian.Uint32(buf[3:7])

	length := binary.BigEndian.Uint32(buf[7:11])
	payload := make([]byte, length)
	if length > 0 {
		_, err := io.ReadFull(rdr, payload)
		if err != nil {
			g.L.Warn("Decode:io.ReadFull", zap.String("err", err.Error()))
			return err
		}
		// 解压
		if t.IsCompress == constant.TypeOfCompressYes {
			t.Payload, err = snappy.Decode(nil, payload)
			if err != nil {
				g.L.Warn("Decode:snappy.Decode", zap.String("error", err.Error()))
				return err
			}
		} else {
			t.Payload = payload
		}
		t.Len = uint32(len(t.Payload))
	}
	return nil
}
