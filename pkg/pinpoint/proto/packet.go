package proto

import (
	"io"
	"net"

	"github.com/bsed/trace/pkg/pinpoint/thrift/trace"
)

const TCP_MAX_PACKET_SIZE int = 16 * 1024
const UDP_MAX_PACKET_SIZE int = 5 * 1024

// packetType
var (
	APPLICATION_SEND           int16 = 1
	APPLICATION_TRACE_SEND     int16 = 2
	APPLICATION_TRACE_SEND_ACK int16 = 3

	APPLICATION_REQUEST  int16 = 5
	APPLICATION_RESPONSE int16 = 6

	APPLICATION_STREAM_CREATE         int16 = 10
	APPLICATION_STREAM_CREATE_SUCCESS int16 = 12
	APPLICATION_STREAM_CREATE_FAIL    int16 = 14

	APPLICATION_STREAM_CLOSE int16 = 15

	APPLICATION_STREAM_PING int16 = 17
	APPLICATION_STREAM_PONG int16 = 18

	APPLICATION_STREAM_RESPONSE int16 = 20

	CONTROL_CLIENT_CLOSE int16 = 100
	CONTROL_SERVER_CLOSE int16 = 110

	// control packet
	CONTROL_HANDSHAKE          int16 = 150
	CONTROL_HANDSHAKE_RESPONSE int16 = 151

	// keep stay because of performance in case of ping and pong. others removed.
	// CONTROL_PING will be deprecated. caused : Two payload types are used in one control packet.
	// since 1.7.0, use CONTROL_PING_SIMPLE, CONTROL_PING_PAYLOAD
	//@Deprecated
	CONTROL_PING int16 = 200
	CONTROL_PONG int16 = 201

	CONTROL_PING_SIMPLE  int16 = 210
	CONTROL_PING_PAYLOAD int16 = 211

	UNKNOWN int16 = 500

	PACKET_TYPE_SIZE int16 = 2
)

type HandshakeResponseCode struct {
	Code        int
	SubCode     int
	CodeMessage string
}

var (
	CODE     string = "code"
	SUB_CODE string = "subCode"
	CLUSTER  string = "cluster"
)

var (
	HANDSHAKE_SUCCESS               = &HandshakeResponseCode{0, 0, "Success."}
	HANDSHAKE_SIMPLEX_COMMUNICATION = &HandshakeResponseCode{0, 1, "Simplex Connection successfully established."}
	HANDSHAKE_DUPLEX_COMMUNICATION  = &HandshakeResponseCode{0, 2, "Duplex Connection successfully established."}

	HANDSHAKE_ALREADY_KNOWN                 = &HandshakeResponseCode{1, 0, "Already Known."}
	HANDSHAKE_ALREADY_SIMPLEX_COMMUNICATION = &HandshakeResponseCode{1, 1, "Already Simplex Connection established."}
	HANDSHAKE_ALREADY_DUPLEX_COMMUNICATION  = &HandshakeResponseCode{1, 2, "Already Duplex Connection established."}

	HANDSHAKE_PROPERTY_ERROR = &HandshakeResponseCode{2, 0, "Property error."}

	HANDSHAKE_PROTOCOL_ERROR = &HandshakeResponseCode{3, 0, "Illegal protocol error."}
	HANDSHAKE_UNKNOWN_ERROR  = &HandshakeResponseCode{4, 0, "Unknown Error."}
	HANDSHAKE_UNKNOWN_CODE   = &HandshakeResponseCode{-1, -1, "Unknown Code."}
)

type Packet interface {
	Decode(conn net.Conn, reader io.Reader) error
	Encode() ([]byte, error)
	GetPacketType() int16
	GetPayload() []byte
	GetRequestID() int
}

func DealRequestResponse(message Packet) *trace.TResult_ {
	//tStruct := thrift.Deserialize(message.GetPayload())
	//isSuccess := false
	//switch m := tStruct.(type) {
	//case *pinpoint.TAgentInfo:
	//
	//	break
	//case *trace.TSqlMetaData:
	//
	//	break
	//case *trace.TApiMetaData:
	//
	//	break
	//case *trace.TStringMetaData:
	//
	//	break
	//default:
	//	g.L.Warn("unknown type", zap.Any("data", m))
	//}
	//isSuccess := true
	result := trace.NewTResult_()
	result.Success = true
	return result
}
