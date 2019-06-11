package thrift

import (
	"encoding/binary"
	"sync"

	"git.apache.org/thrift.git/lib/go/thrift"
)

type DeserializeServer struct {
	trans      *thrift.TMemoryBuffer
	dsprotocol thrift.TProtocol
}

func NewDeserializeServer() *DeserializeServer {
	var trans = thrift.NewTMemoryBuffer()
	var dsprotocol = thrift.NewTCompactProtocolFactory().GetProtocol(trans)
	return &DeserializeServer{trans: trans, dsprotocol: dsprotocol}
}

var deserializePool sync.Pool

func Deserialize(payload []byte) thrift.TStruct {
	var dServer *DeserializeServer
	v := deserializePool.Get()
	if v == nil {
		dServer = NewDeserializeServer()
	} else {
		dServer = v.(*DeserializeServer)
	}

	dServer.trans.Reset()
	_, err := dServer.trans.Write(payload)
	if err != nil {
		return nil
	}
	header := readHeader(dServer.dsprotocol)
	if validate(header) {
		tStruct := TBaseLookup(header.Type)
		tStruct.Read(dServer.dsprotocol)
		deserializePool.Put(dServer)
		return tStruct
	} else {
		deserializePool.Put(dServer)
		return nil
	}
}

func validate(header *Header) bool {
	if header.Signature == HEADER_SIGNATURE {
		return true
	}
	return false
}

func readHeader(dsprotocol thrift.TProtocol) *Header {
	Signature, err0 := readByte(dsprotocol)
	Version, err1 := readByte(dsprotocol)
	byte1, err2 := readByte(dsprotocol)
	byte2, err3 := readByte(dsprotocol)
	if err0 != nil || err1 != nil || err2 != nil || err3 != nil {
		return nil
	}
	Type := bytesToInt16(byte1, byte2)
	return &Header{
		Signature: Signature,
		Version:   Version,
		Type:      Type,
	}
}

func bytesToInt16(byte1, byte2 byte) int16 {
	buf := make([]byte, 2)
	buf[0] = byte1
	buf[1] = byte2
	return int16(binary.BigEndian.Uint16(buf))
}

func readByte(dsprotocol thrift.TProtocol) (byte, error) {
	buf, err := dsprotocol.ReadByte()
	if err != nil {
		return byte(buf), err
	}
	return byte(buf), nil
}
