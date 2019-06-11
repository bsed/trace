package thrift

import (
	"encoding/binary"
	"sync"

	"git.apache.org/thrift.git/lib/go/thrift"
)

type SerializeServer struct {
	baos      *thrift.TMemoryBuffer
	sprotocol thrift.TProtocol
}

func NewSerializeServer() *SerializeServer {
	var baos = thrift.NewTMemoryBufferLen(65507)
	var sprotocol = thrift.NewTCompactProtocolFactory().GetProtocol(baos)
	return &SerializeServer{baos: baos, sprotocol: sprotocol}
}

var serializePool sync.Pool

func Serialize(tStruct thrift.TStruct) []byte {
	var sServer *SerializeServer
	v := serializePool.Get()
	if v == nil {
		sServer = NewSerializeServer()
	} else {
		sServer = v.(*SerializeServer)
	}
	header := HeaderLookup(tStruct)
	sServer.baos.Reset()
	writeHeader(sServer.sprotocol, header)
	tStruct.Write(sServer.sprotocol)
	buf := sServer.baos.Bytes()
	serializePool.Put(sServer)
	return buf
}

func SerializeNew(tStruct thrift.TStruct) []byte {
	var sServer *SerializeServer = NewSerializeServer()
	header := HeaderLookup(tStruct)
	sServer.baos.Reset()
	writeHeader(sServer.sprotocol, header)
	tStruct.Write(sServer.sprotocol)
	buf := sServer.baos.Bytes()
	return buf
}

func writeHeader(sprotocol thrift.TProtocol, header *Header) {
	sprotocol.WriteByte(int8(header.Signature))
	sprotocol.WriteByte(int8(header.Version))
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, uint16(header.Type))
	sprotocol.WriteByte(int8(buf[0]))
	sprotocol.WriteByte(int8(buf[1]))
}
