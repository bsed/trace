package thrift

import (
	"git.apache.org/thrift.git/lib/go/thrift"
	"github.com/bsed/trace/pkg/pinpoint/thrift/command"
	"github.com/bsed/trace/pkg/pinpoint/thrift/pinpoint"
	"github.com/bsed/trace/pkg/pinpoint/thrift/trace"
)

type Header struct {
	Signature byte
	Version   byte
	Type      int16
}

const (
	NETWORK_CHECK    int16 = 10
	SPAN             int16 = 40
	AGENT_INFO       int16 = 50
	AGENT_STAT       int16 = 55
	AGENT_STAT_BATCH int16 = 56
	SPANCHUNK        int16 = 70
	SPANEVENT        int16 = 80
	SQLMETADATA      int16 = 300
	APIMETADATA      int16 = 310
	RESULT           int16 = 320
	STRINGMETADATA   int16 = 330
	CHUNK            int16 = 400

	//####################command#####################
	TRANSFER                          int16 = 700
	TRANSFER_RESPONSE                 int16 = 701
	ECHO                              int16 = 710
	THREAD_DUMP                       int16 = 720
	THREAD_DUMP_RESPONSE              int16 = 721
	ACTIVE_THREAD_COUNT               int16 = 730
	ACTIVE_THREAD_COUNT_RESPONSE      int16 = 731
	ACTIVE_THREAD_DUMP                int16 = 740
	ACTIVE_THREAD_DUMP_RESPONSE       int16 = 741
	ACTIVE_THREAD_LIGHT_DUMP          int16 = 750
	ACTIVE_THREAD_LIGHT_DUMP_RESPONSE int16 = 751

	ACTIVE_COMMAND_HEAP          int16 = 780
	ACTIVE_COMMAND_HEAP_RESPONSE int16 = 781
	//####################command end#####################

	HEADER_SIGNATURE byte = 0xef
)

func NewHeader(Type int16) *Header {
	return &Header{
		Signature: HEADER_SIGNATURE,
		Version:   0x10,
		Type:      Type}
}

func HeaderLookup(tStruct thrift.TStruct) *Header {
	switch tStruct.(type) {
	case *trace.TSpan:
		return NewHeader(SPAN)
	case *trace.TSpanChunk:
		return NewHeader(SPANCHUNK)
	case *trace.TSpanEvent:
		return NewHeader(SPANEVENT)
	case *pinpoint.TAgentInfo:
		return NewHeader(AGENT_INFO)
	case *pinpoint.TAgentStat:
		return NewHeader(AGENT_STAT)
	case *pinpoint.TAgentStatBatch:
		return NewHeader(AGENT_STAT_BATCH)
	case *trace.TSqlMetaData:
		return NewHeader(SQLMETADATA)
	case *trace.TApiMetaData:
		return NewHeader(APIMETADATA)
	case *trace.TResult_:
		return NewHeader(RESULT)
	case *trace.TStringMetaData:
		return NewHeader(STRINGMETADATA)
	case *command.TCommandTransfer:
		return NewHeader(TRANSFER)
	case *command.TCommandTransferResponse:
		return NewHeader(TRANSFER_RESPONSE)
	case *command.TCommandEcho:
		return NewHeader(ECHO)
	case *command.TCommandThreadDump:
		return NewHeader(THREAD_DUMP)
	case *command.TCommandThreadDumpResponse:
		return NewHeader(THREAD_DUMP_RESPONSE)
	case *command.TCmdActiveThreadCount:
		return NewHeader(ACTIVE_THREAD_COUNT)
	case *command.TCmdActiveThreadCountRes:
		return NewHeader(ACTIVE_THREAD_COUNT_RESPONSE)
	case *command.TCmdActiveThreadDump:
		return NewHeader(ACTIVE_THREAD_DUMP)
	case *command.TCmdActiveThreadDumpRes:
		return NewHeader(ACTIVE_THREAD_DUMP_RESPONSE)
	case *command.TCmdActiveThreadLightDump:
		return NewHeader(ACTIVE_THREAD_LIGHT_DUMP)
	case *command.TCmdActiveThreadLightDumpRes:
		return NewHeader(ACTIVE_THREAD_LIGHT_DUMP_RESPONSE)
		// case *command.TCommandHeap:
		// 	return NewHeader(ACTIVE_COMMAND_HEAP)
		// case *command.TCommandHeapResponse:
		// 	return NewHeader(ACTIVE_COMMAND_HEAP_RESPONSE)
	}
	return nil
}

func TBaseLookup(Type int16) thrift.TStruct {
	switch Type {
	case SPAN:
		return trace.NewTSpan()
	case AGENT_INFO:
		return pinpoint.NewTAgentInfo()
	case AGENT_STAT:
		return pinpoint.NewTAgentStat()
	case AGENT_STAT_BATCH:
		return pinpoint.NewTAgentStatBatch()
	case SPANCHUNK:
		return trace.NewTSpanChunk()
	case SPANEVENT:
		return trace.NewTSpanEvent()
	case SQLMETADATA:
		return trace.NewTSqlMetaData()
	case APIMETADATA:
		return trace.NewTApiMetaData()
	case RESULT:
		return trace.NewTResult_()
	case STRINGMETADATA:
		return trace.NewTStringMetaData()
	case TRANSFER:
		return command.NewTCommandTransfer()
	case TRANSFER_RESPONSE:
		return command.NewTCommandTransferResponse()
	case ECHO:
		return command.NewTCommandEcho()
	case THREAD_DUMP:
		return command.NewTCommandThreadDump()
	case THREAD_DUMP_RESPONSE:
		return command.NewTCommandThreadDumpResponse()
	case ACTIVE_THREAD_COUNT:
		return command.NewTCmdActiveThreadCount()
	case ACTIVE_THREAD_COUNT_RESPONSE:
		return command.NewTCmdActiveThreadCountRes()
	case ACTIVE_THREAD_DUMP:
		return command.NewTCmdActiveThreadDump()
	case ACTIVE_THREAD_DUMP_RESPONSE:
		return command.NewTCmdActiveThreadDumpRes()
	case ACTIVE_THREAD_LIGHT_DUMP:
		return command.NewTCmdActiveThreadLightDump()
	case ACTIVE_THREAD_LIGHT_DUMP_RESPONSE:
		return command.NewTCmdActiveThreadLightDumpRes()
		// case ACTIVE_COMMAND_HEAP:
		// 	return command.NewTCommandHeap()
		// case ACTIVE_COMMAND_HEAP_RESPONSE:
		// 	return command.NewTCommandHeapResponse()
	}
	return nil
}
