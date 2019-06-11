package service

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/bsed/trace/pkg/constant"
	"github.com/bsed/trace/pkg/network"
	"github.com/bsed/trace/pkg/pinpoint/thrift"
	"github.com/bsed/trace/pkg/pinpoint/thrift/pinpoint"
	"github.com/bsed/trace/pkg/pinpoint/thrift/trace"
	"github.com/vmihailenco/msgpack"
	"go.uber.org/zap"
)

// pinpointPacket 处理agent上报的监控数据
func (t *tcpClient) pinpointPacket(conn net.Conn, tracePack *network.TracePack) error {
	packet := &network.SpansPacket{}
	if err := msgpack.Unmarshal(tracePack.Payload, packet); err != nil {
		logger.Warn("msgpack.Unmarshal", zap.String("error", err.Error()))
		return err
	}

	switch packet.Type {
	case constant.TypeOfTCPData:
		for _, value := range packet.Payload {
			switch value.Type {
			case constant.TypeOfRegister:
				agentInfo := network.NewAgentInfo()
				if err := msgpack.Unmarshal(value.Spans, agentInfo); err != nil {
					logger.Warn("msgpack Unmarshal", zap.String("error", err.Error()))
					return err
				}
				if err := gCollector.storage.AppNameStore(agentInfo.AppName); err != nil {
					logger.Warn("insert apps error", zap.String("error", err.Error()))
					return err
				}

				if err := gCollector.storage.AgentStore(agentInfo, true); err != nil {
					logger.Warn("agent Store", zap.String("error", err.Error()))
					return err
				}

				t.appName = agentInfo.AppName
				t.agentID = agentInfo.AgentID

				logger.Info("Online", zap.String("appName", agentInfo.AppName), zap.String("agentID", agentInfo.AgentID))
				// 注册信息原样返回
				if _, err := conn.Write(tracePack.Encode()); err != nil {
					logger.Warn("conn.Write", zap.String("error", err.Error()))
					return err
				}

				break
			case constant.TypeOfAgentOffline:
				agentInfo := network.NewAgentInfo()
				if err := msgpack.Unmarshal(value.Spans, agentInfo); err != nil {
					logger.Warn("msgpack.Unmarshal", zap.String("error", err.Error()))
					return err
				}

				if err := gCollector.storage.UpdateAgentState(agentInfo.AppName, agentInfo.AgentID, false); err != nil {
					logger.Warn("update agent state Store", zap.String("error", err.Error()))
					return err
				}

				// 信息原样返回
				if _, err := conn.Write(tracePack.Encode()); err != nil {
					logger.Warn("conn.Write", zap.String("error", err.Error()))
					return err
				}
				logger.Info("Offline", zap.String("appName", agentInfo.AppName), zap.String("agentID", agentInfo.AgentID))

				break
			case constant.TypeOfAgentInfo, constant.TypeOfSQLMetaData, constant.TypeOfAPIMetaData, constant.TypeOfStringMetaData:
				if err := t.tcpRequestResponse(packet, value.Spans); err != nil {
					logger.Warn("tcpRequestResponse", zap.String("error", err.Error()))
					return err
				}
				break
			default:
				logger.Warn("unknow type", zap.String("type", fmt.Sprintf("%T", value.Type)), zap.Uint16("type", value.Type))
				break
			}
		}
		break
	case constant.TypeOfUDPData:
		for _, value := range packet.Payload {
			t.udpRequest(packet.AppName, packet.AgentID, value.Spans)
		}
		break
	default:
		logger.Warn("unknow type", zap.String("type", fmt.Sprintf("%T", packet.Type)), zap.Uint16("value", packet.Type))
	}
	return nil
}

// udpRequest udp报文处理
func (t *tcpClient) udpRequest(appName, agentID string, data []byte) {
	tStruct := thrift.Deserialize(data)
	switch m := tStruct.(type) {
	case *trace.TSpan:
		gCollector.storage.SpanStore(m)
		gCollector.apps.routerSapn(appName, agentID, m)
		break
	case *trace.TSpanChunk:
		gCollector.storage.SpanChunkStore(m)
		gCollector.apps.routersapnChunk(appName, agentID, m)
		break
	case *pinpoint.TAgentStat:
		gCollector.apps.online(appName, agentID)
		gCollector.apps.routerAgentStat(appName, agentID, m)
		if err := gCollector.storage.WriteAgentStat(appName, agentID, m, data); err != nil {
			logger.Warn("agent stat", zap.String("error", err.Error()))
		}
		break
	case *pinpoint.TAgentStatBatch:
		gCollector.apps.online(appName, agentID)
		gCollector.apps.routerAgentStatBatch(appName, agentID, m)
		if err := gCollector.storage.WriteAgentStatBatch(appName, agentID, m, data); err != nil {
			logger.Warn("stat batch", zap.String("error", err.Error()))
		}
		break
	default:
		logger.Warn("unknow type", zap.String("type", fmt.Sprintf("%T", m)))
	}
}

// tcpRequestResponse ...
func (t *tcpClient) tcpRequestResponse(packet *network.SpansPacket, message []byte) error {
	tStruct := thrift.Deserialize(message)
	switch m := tStruct.(type) {
	case *pinpoint.TAgentInfo:
		agentInfo, err := json.Marshal(m)
		if err != nil {
			logger.Warn("json.Marshal", zap.String("error", err.Error()))
			return err
		}
		if err := gCollector.storage.AgentInfoStore(packet.AppName, packet.AgentID, m.StartTimestamp, agentInfo); err != nil {
			logger.Warn("agent info store", zap.String("error", err.Error()))
			return err
		}
		break
	case *trace.TSqlMetaData:
		if err := gCollector.storage.AppSQLStore(packet.AppName, m); err != nil {
			logger.Warn("sql store", zap.String("error", err.Error()))
			return err
		}
		break
	case *trace.TApiMetaData:
		if err := gCollector.storage.AppMethodStore(packet.AppName, m); err != nil {
			logger.Warn("api store", zap.String("error", err.Error()))
			return err
		}
		break
	case *trace.TStringMetaData:
		if err := gCollector.storage.AppStringStore(packet.AppName, m); err != nil {
			logger.Warn("string store", zap.String("error", err.Error()))
			return err
		}
		break
	default:
		logger.Warn("unknown type", zap.String("type", fmt.Sprintf("%t", m)))
	}
	return nil
}
