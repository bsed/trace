package service

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/bsed/trace/agent/misc"
	"github.com/bsed/trace/pkg/constant"
	"github.com/bsed/trace/pkg/network"
	"github.com/bsed/trace/pkg/pinpoint/proto"
	"github.com/bsed/trace/pkg/pinpoint/thrift"
	"github.com/bsed/trace/pkg/pinpoint/thrift/pinpoint"
	"github.com/bsed/trace/pkg/pinpoint/thrift/trace"
	"github.com/vmihailenco/msgpack"
	"go.uber.org/zap"
)

// Pinpoint p数据采集
type Pinpoint struct {
	tcpChan chan *network.Spans // tcp报文接收管道
	udpChan chan *network.Spans // udp报文接收管道
}

func newPinpoint() *Pinpoint {
	return &Pinpoint{
		tcpChan: make(chan *network.Spans, 100),
		udpChan: make(chan *network.Spans, 300),
	}
}

// Start 启动pinpoint采集服务
func (p *Pinpoint) Start() error {
	// 接收pinpoint三种数据
	go p.AgentInfo()
	go p.AgentStat()
	go p.AgentSpan()

	// pinpoint tcp、udp信息采集&上报
	go p.tcpCollector()
	go p.udpCollector()
	return nil
}

// Close 关闭pinpoint采集服务
func (p *Pinpoint) Close() error {
	return nil
}

// AgentInfo ...
func (p *Pinpoint) AgentInfo() error {
	l, err := net.Listen("tcp", misc.Conf.Pinpoint.InfoAddr)
	if err != nil {
		logger.Fatal("AgentInfo", zap.Error(err))
		return err
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			logger.Fatal("AgentInfo", zap.String("addr", misc.Conf.Pinpoint.InfoAddr), zap.Error(err))
			return err
		}
		go p.agentInfo(conn)
	}
}

// AgentStat ...
func (p *Pinpoint) AgentStat() error {
	addrInfo, _ := net.ResolveUDPAddr("udp", misc.Conf.Pinpoint.StatAddr)
	listener, err := net.ListenUDP("udp", addrInfo)
	if err != nil {
		logger.Fatal("AgentStat ListenUDP", zap.String("addr", misc.Conf.Pinpoint.StatAddr), zap.String("error", err.Error()))
	}

	for {
		data := make([]byte, proto.UDP_MAX_PACKET_SIZE)
		listener.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, _, err := listener.ReadFrom(data)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {

			} else {
				logger.Warn("AgentStat ReadFrom", zap.String("addr", misc.Conf.Pinpoint.StatAddr), zap.String("error", err.Error()))
			}
			continue
		}
		if n == 0 {
			continue
		}

		spans, err := udpRead(data[:n])
		if err != nil {
			logger.Warn("udpRead", zap.String("error", err.Error()))
			return err
		}
		p.udpChan <- spans
	}
}

// AgentSpan ...
func (p *Pinpoint) AgentSpan() error {
	addrInfo, _ := net.ResolveUDPAddr("udp", misc.Conf.Pinpoint.SpanAddr)
	listener, err := net.ListenUDP("udp", addrInfo)
	if err != nil {
		logger.Fatal("listen udp", zap.String("addr", misc.Conf.Pinpoint.SpanAddr), zap.String("error", err.Error()))
	}

	for {
		data := make([]byte, proto.UDP_MAX_PACKET_SIZE)
		listener.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, _, err := listener.ReadFrom(data)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {

			} else {
				logger.Warn("readfrom", zap.String("addr", misc.Conf.Pinpoint.SpanAddr), zap.String("error", err.Error()))
			}
			continue
		}
		if n == 0 {
			continue
		}

		spans, err := udpRead(data[:n])
		if err != nil {
			logger.Warn("udpRead", zap.String("error", err.Error()))
			return err
		}
		p.udpChan <- spans
	}
}

// tcpCollector ...
func (p *Pinpoint) tcpCollector() {

	spanPack := network.NewSpansPacket()
	for {
		select {
		case span, ok := <-p.tcpChan:
			if !ok {
				break
			}

			spanPack.Type = constant.TypeOfTCPData
			spanPack.AppName = gAgent.appName
			spanPack.AgentID = gAgent.agentID

			tracePack := &network.TracePack{
				Type:       constant.TypeOfPinpoint,
				IsSync:     constant.TypeOfSyncNo,
				IsCompress: constant.TypeOfCompressYes,
			}

			spanPack.Payload = append(spanPack.Payload, span)
			payload, err := msgpack.Marshal(spanPack)
			if err != nil {
				logger.Warn("msgpack Marshal", zap.String("error", err.Error()))
				// 清空缓存
				spanPack.Payload = spanPack.Payload[:0]
				break
			}
			tracePack.Payload = payload
			// 发送
			if err := gAgent.collector.write(tracePack); err != nil {
				logger.Warn("write", zap.String("error", err.Error()))
			}
			// 清空缓存
			spanPack.Payload = spanPack.Payload[:0]
			break
		}
	}
}

// udpCollector ...
func (p *Pinpoint) udpCollector() {
	// 定时器
	ticker := time.NewTicker(time.Duration(misc.Conf.Pinpoint.SpanReportInterval) * time.Millisecond)
	defer ticker.Stop()
	spanPack := network.NewSpansPacket()

	for {
		select {
		case spanData, ok := <-p.udpChan:
			if ok {
				spanPack.Type = constant.TypeOfUDPData
				spanPack.AppName = gAgent.appName
				spanPack.AgentID = gAgent.agentID

				tracePack := &network.TracePack{
					Type:       constant.TypeOfPinpoint,
					IsSync:     constant.TypeOfSyncNo,
					IsCompress: constant.TypeOfCompressYes,
				}

				spanPack.Payload = append(spanPack.Payload, spanData)
				if len(spanPack.Payload) >= misc.Conf.Pinpoint.SpanQueueLen {
					payload, err := msgpack.Marshal(spanPack)
					if err != nil {
						logger.Warn("msgpack Marshal", zap.String("error", err.Error()))
						// 清空缓存
						spanPack.Payload = spanPack.Payload[:0]
						continue
					}
					tracePack.Payload = payload
					// 发送
					if err := gAgent.collector.write(tracePack); err != nil {
						logger.Warn("write", zap.String("error", err.Error()))
					}
					// 清空缓存
					spanPack.Payload = spanPack.Payload[:0]
				}
			}
			break
		case <-ticker.C:
			if len(spanPack.Payload) > 0 {

				spanPack.Type = constant.TypeOfUDPData
				spanPack.AppName = gAgent.appName
				spanPack.AgentID = gAgent.agentID

				tracePack := &network.TracePack{
					Type:       constant.TypeOfPinpoint,
					IsSync:     constant.TypeOfSyncNo,
					IsCompress: constant.TypeOfCompressYes,
				}

				payload, err := msgpack.Marshal(spanPack)
				if err != nil {
					logger.Warn("agentInfo:msgpack.Marshal", zap.String("error", err.Error()))
					// 清空缓存
					spanPack.Payload = spanPack.Payload[:0]
					continue
				}
				tracePack.Payload = payload
				// 发送
				if err := gAgent.collector.write(tracePack); err != nil {
					logger.Warn("write", zap.String("error", err.Error()))
				}
				// 清空缓存
				spanPack.Payload = spanPack.Payload[:0]
			}
		}
	}
}

func (p *Pinpoint) agentInfo(conn net.Conn) error {
	defer func() {
		if err := recover(); err != nil {
			return
		}
	}()

	isRecvOffline := false
	defer func() {
		updateAgentStats(false)
		if !isRecvOffline {
			// sdk客户端断线
			gAgent.isLive = false
		}
	}()

	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()
	isSendOnline := false
	reader := bufio.NewReaderSize(conn, proto.TCP_MAX_PACKET_SIZE)
	buf := make([]byte, 2)
	for {
		// response packet
		var rePacket proto.Packet
		var err error
		// response flag
		isRePacket := false
		if _, err := io.ReadFull(reader, buf); err != nil {
			logger.Error("io.ReadFull", zap.Error(err))
			return err
		}
		// read packet type
		packetType := int16(binary.BigEndian.Uint16(buf[0:2]))
		switch packetType {
		case proto.APPLICATION_SEND:
			logger.Debug("agentInfo", zap.String("type", "APPLICATION_SEND"))
			applicationSend := proto.NewApplicationSend()
			if err := applicationSend.Decode(conn, reader); err != nil {
				logger.Warn("applicationSend.Decode", zap.String("error", err.Error()))
				return err
			}

			spans, err := handletcp(applicationSend.GetPayload())
			if err != nil {
				logger.Warn("handletcp", zap.String("error", err.Error()))
				return err
			}

			p.tcpChan <- spans
			break

		case proto.APPLICATION_REQUEST:
			// logger.Debug("agentInfo", zap.String("type", "APPLICATION_REQUEST"))
			applicationRequest := proto.NewApplicationRequest()
			if err := applicationRequest.Decode(conn, reader); err != nil {
				logger.Warn("decode", zap.String("error", err.Error()))
				return err
			}

			spans, err := handletcp(applicationRequest.GetPayload())
			if err != nil {
				logger.Warn("handle tcp", zap.String("error", err.Error()))
				return err
			}
			p.tcpChan <- spans

			tResult := proto.DealRequestResponse(applicationRequest)
			response := proto.NewApplicationResponse()
			response.RequestID = applicationRequest.GetRequestID()
			response.Payload = thrift.Serialize(tResult)
			isRePacket = true
			rePacket = response
			break

		case proto.APPLICATION_RESPONSE:
			logger.Debug("agentInfo", zap.String("type", "APPLICATION_RESPONSE"))
			applicationResponse := proto.NewApplicationResponse()
			if err := applicationResponse.Decode(conn, reader); err != nil {
				logger.Warn("response Decode", zap.String("error", err.Error()))
				return err
			}
			break

		case proto.APPLICATION_STREAM_CREATE:
			logger.Debug("agentInfo", zap.String("type", "APPLICATION_STREAM_CREATE"))
			applicationStreamCreate := proto.NewApplicationStreamCreate()
			if err := applicationStreamCreate.Decode(conn, reader); err != nil {
				logger.Warn("stream create Decode", zap.String("error", err.Error()))
				return err
			}
			break

		case proto.APPLICATION_STREAM_CLOSE:
			logger.Debug("agentInfo", zap.String("type", "APPLICATION_STREAM_CLOSE"))
			applicationStreamClose := proto.NewApplicationStreamClose()
			if err := applicationStreamClose.Decode(conn, reader); err != nil {
				logger.Warn("stream close Decode", zap.String("error", err.Error()))
				return err
			}
			break

		case proto.APPLICATION_STREAM_CREATE_SUCCESS:
			logger.Debug("agentInfo", zap.String("type", "APPLICATION_STREAM_CREATE_SUCCESS"))
			applicationStreamCreateSuccess := proto.NewApplicationStreamCreateSuccess()
			if err := applicationStreamCreateSuccess.Decode(conn, reader); err != nil {
				logger.Warn("stream create success Decode", zap.String("error", err.Error()))
				return err
			}
			break

		case proto.APPLICATION_STREAM_CREATE_FAIL:
			logger.Debug("agentInfo", zap.String("type", "APPLICATION_STREAM_CREATE_FAIL"))
			applicationStreamCreateFail := proto.NewApplicationStreamCreateFail()
			if err := applicationStreamCreateFail.Decode(conn, reader); err != nil {
				logger.Warn("stream create fail Decode", zap.String("error", err.Error()))
				return err
			}

			break

		case proto.APPLICATION_STREAM_RESPONSE:
			logger.Debug("agentInfo", zap.String("type", "APPLICATION_STREAM_RESPONSE"))
			applicationStreamResponse := proto.NewApplicationStreamResponse()
			if err := applicationStreamResponse.Decode(conn, reader); err != nil {
				logger.Warn("stream response decode", zap.String("error", err.Error()))
				return err
			}
			break

		case proto.APPLICATION_STREAM_PING:
			logger.Debug("agentInfo", zap.String("type", "APPLICATION_STREAM_PING"))
			applicationStreamPing := proto.NewApplicationStreamPing()
			if err := applicationStreamPing.Decode(conn, reader); err != nil {
				logger.Warn("stream ping decode", zap.String("error", err.Error()))
				return err
			}
			break

		case proto.APPLICATION_STREAM_PONG:
			logger.Debug("agentInfo", zap.String("type", "APPLICATION_STREAM_PONG"))
			applicationStreamPong := proto.NewApplicationStreamPong()
			if err := applicationStreamPong.Decode(conn, reader); err != nil {
				logger.Warn("stream pong decode", zap.String("error", err.Error()))
				return err
			}
			break

		case proto.CONTROL_HANDSHAKE:
			// 获取并保存Agent信息
			logger.Debug("agentInfo", zap.String("type", "CONTROL_HANDSHAKE"))
			controlHandShake := proto.NewControlHandShake()
			if err := controlHandShake.Decode(conn, reader); err != nil {
				logger.Warn("control hand shake decode", zap.String("error", err.Error()))
				return err
			}

			agentInfo := network.NewAgentInfo()
			if err := json.Unmarshal(controlHandShake.GetPayload(), agentInfo); err != nil {
				logger.Warn("json unmarshal", zap.String("error", err.Error()))
				return err
			}

			logger.Debug("agentInfo", zap.String("name", agentInfo.AppName), zap.String("id", agentInfo.AgentID))

			// 保存App信息
			gAgent.appName = agentInfo.AppName
			gAgent.agentID = agentInfo.AgentID
			gAgent.agentInfo.AppName = agentInfo.AppName
			gAgent.agentInfo.AgentID = agentInfo.AgentID
			gAgent.agentInfo.ServiceType = agentInfo.ServiceType
			gAgent.agentInfo.HostName = agentInfo.HostName
			gAgent.agentInfo.IP4S = agentInfo.IP4S
			gAgent.agentInfo.StartTimestamp = agentInfo.StartTimestamp
			gAgent.agentInfo.EndTimestamp = agentInfo.EndTimestamp
			gAgent.agentInfo.IsContainer = agentInfo.IsContainer
			gAgent.agentInfo.OperatingEnv = agentInfo.OperatingEnv
			gAgent.agentInfo.AgentInfo = agentInfo.AgentInfo

			if !isSendOnline {
				if err := updateAgentStats(true); err != nil {
					logger.Warn("agent update stats", zap.String("error", err.Error()), zap.Bool("live", true))
					return err
				}
				isSendOnline = true
			}

			rePacket, err = createResponse(controlHandShake)
			if err != nil {
				logger.Warn("createResponse", zap.String("error", err.Error()))
				return err
			}

			break

		case proto.CONTROL_CLIENT_CLOSE:
			logger.Debug("agentInfo", zap.String("type", "CONTROL_CLIENT_CLOSE"))
			if err := updateAgentStats(false); err != nil {
				logger.Warn("agent update stats", zap.String("error", err.Error()), zap.Bool("live", false))
				return err
			}

			isSendOnline = false
			break

		case proto.CONTROL_PING:
			logger.Debug("agentInfo", zap.String("type", "CONTROL_PING"))
			controlPing := proto.NewControlPing()
			if err := controlPing.Decode(conn, reader); err != nil {
				logger.Warn("controlPing.Decode", zap.String("error", err.Error()))
				return err
			}
			isRePacket = true
			controlPong := proto.NewControlPong()
			rePacket = controlPong
			break

		case proto.CONTROL_PING_SIMPLE:
			logger.Debug("agentInfo", zap.String("type", "CONTROL_PING_SIMPLE"))
			isRePacket = true
			controlPong := proto.NewControlPong()
			rePacket = controlPong
			break

		case proto.CONTROL_PING_PAYLOAD:
			logger.Debug("agentInfo", zap.String("type", "CONTROL_PING"))
			controlPingPayload := proto.NewControlPing()
			if err := controlPingPayload.Decode(conn, reader); err != nil {
				logger.Warn("control ping payload decode", zap.String("error", err.Error()))
				return err
			}
			break

		default:
			logger.Warn("unaware packet Type", zap.Int16("packetType", packetType))
		}

		if isRePacket {
			body, err := rePacket.Encode()
			if err != nil {
				logger.Warn("rePacket encode", zap.String("error", err.Error()))
				return err
			}

			if _, err := conn.Write(body); err != nil {
				logger.Warn("write", zap.String("error", err.Error()))
				return err
			}
		}
	}
}

func updateAgentStats(islive bool) error {
	spanPackets := network.NewSpansPacket()
	spanPackets.Type = constant.TypeOfTCPData
	spanPackets.AppName = gAgent.appName
	spanPackets.AgentID = gAgent.agentID

	agentInfo, err := msgpack.Marshal(gAgent.agentInfo)
	if err != nil {
		logger.Warn("msgpack Marshal", zap.String("error", err.Error()))
		return err
	}
	spans := &network.Spans{
		Spans: agentInfo,
	}
	if islive == false {
		spans.Type = constant.TypeOfAgentOffline
	} else {
		spans.Type = constant.TypeOfRegister
	}

	spanPackets.Payload = append(spanPackets.Payload, spans)
	payload, err := msgpack.Marshal(spanPackets)
	if err != nil {
		logger.Warn("msgpack Marshal", zap.String("error", err.Error()))
		return err
	}

	id := gAgent.getSyncID()
	tracePacket := &network.TracePack{
		Type:       constant.TypeOfPinpoint,
		IsSync:     constant.TypeOfSyncYes,
		IsCompress: constant.TypeOfCompressNo,
		ID:         id,
		Payload:    payload,
	}

	if err := gAgent.collector.write(tracePacket); err != nil {
		logger.Warn("write info", zap.String("error", err.Error()))
		return err
	}

	// 创建chan
	if _, ok := gAgent.syncCall.newChan(id, 10); !ok {
		logger.Warn("syncCall newChan", zap.String("error", "创建sync chan失败"))
		return err
	}

	// 阻塞同步等待，并关闭chan
	if _, err := gAgent.syncCall.syncRead(id, 10, true); err != nil {
		logger.Warn("syncRead", zap.String("error", err.Error()))
		return err
	}
	return nil
}

// handletcp ...
func handletcp(message []byte) (*network.Spans, error) {
	spans := network.NewSpans()
	tStruct := thrift.Deserialize(message)
	switch m := tStruct.(type) {
	case *pinpoint.TAgentInfo:
		spans.Type = constant.TypeOfAgentInfo
		spans.Spans = message
		break
	case *trace.TSqlMetaData:
		spans.Type = constant.TypeOfSQLMetaData
		spans.Spans = message
		break
	case *trace.TApiMetaData:
		spans.Type = constant.TypeOfAPIMetaData
		spans.Spans = message
		break
	case *trace.TStringMetaData:
		spans.Type = constant.TypeOfStringMetaData
		spans.Spans = message
		break
	default:
		logger.Warn("unknown type", zap.String("type", fmt.Sprintf("unknow type [%T]", m)), zap.Any("data", tStruct))
		return nil, fmt.Errorf("unknow type %t", m)
	}
	return spans, nil
}

func createResponse(in proto.Packet) (proto.Packet, error) {
	packType := in.GetPacketType()
	switch packType {
	case proto.CONTROL_HANDSHAKE:
		resultMap := make(map[string]interface{})
		resultMap[proto.CODE] = proto.HANDSHAKE_DUPLEX_COMMUNICATION.Code
		resultMap[proto.SUB_CODE] = proto.HANDSHAKE_DUPLEX_COMMUNICATION.SubCode
		payload, err := json.Marshal(resultMap)
		if err != nil {
			logger.Warn("json.Marshal", zap.String("error", err.Error()))
			return nil, err
		}
		controlHandShakeResponse := proto.NewControlHandShakeResponse()
		controlHandShakeResponse.Payload = payload
		controlHandShakeResponse.RequestID = in.GetRequestID()
		return controlHandShakeResponse, nil

	default:
		break
	}

	return nil, nil
}
