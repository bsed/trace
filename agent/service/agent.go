package service

import (
	"sync/atomic"

	"github.com/bsed/trace/pkg/network"

	"go.uber.org/zap"
)

// Agent ...
type Agent struct {
	appName   string             // 服务名
	agentID   string             // 服务agent ID
	etcd      *Etcd              // 服务发现
	collector *Collector         // 监控指标上报
	pinpoint  *Pinpoint          // pinpoint采集服务
	isLive    bool               // app是否存活
	syncID    uint32             // 同步请求ID
	syncCall  *SyncCall          // 同步请求
	agentInfo *network.AgentInfo // 监控上报的agent info原信息
}

var gAgent *Agent
var logger *zap.Logger

// New new agent
func New(l *zap.Logger) *Agent {
	logger = l
	gAgent = &Agent{
		etcd:      newEtcd(),
		collector: newCollector(),
		pinpoint:  newPinpoint(),
		syncCall:  NewSyncCall(),
		agentInfo: network.NewAgentInfo(),
	}
	return gAgent
}

// Start 启动
func (a *Agent) Start() error {

	// etcd 初始化
	if err := a.etcd.Init(); err != nil {
		logger.Warn("etcd init", zap.String("error", err.Error()))
		return err
	}

	// 启动服务发现
	if err := a.etcd.Start(); err != nil {
		logger.Warn("etcd start", zap.String("error", err.Error()))
		return err
	}

	// 监控采集服务启动
	if err := a.pinpoint.Start(); err != nil {
		logger.Warn("pinpoint start", zap.String("error", err.Error()))
		return err
	}

	// 为agentd提供健康检查
	initHealth()
	// agent 信息上报服务
	// go reportAgentInfo()

	logger.Info("Agent start ok")

	return nil
}

// Close 关闭
func (a *Agent) Close() error {
	return nil
}

// // reportAgentInfo 上报agent 信息
// func reportAgentInfo() {
// 	for {
// 		time.Sleep(1 * time.Second)
// 		if !gAgent.isLive {
// 			continue
// 		}
// 		break
// 	}
// 	for {
// 		time.Sleep(10 * time.Second)
// 		spanPackets := network.NewSpansPacket()
// 		spanPackets.Type = constant.TypeOfTCPData
// 		spanPackets.AppName = gAgent.appName
// 		spanPackets.AgentID = gAgent.agentID

// 		agentInfo, err := msgpack.Marshal(gAgent.agentInfo)
// 		if err != nil {
// 			logger.Warn("msgpack Marshal", zap.String("error", err.Error()))
// 			continue
// 		}
// 		spans := &network.Spans{
// 			Spans: agentInfo,
// 		}
// 		if gAgent.isLive == false {
// 			spans.Type = constant.TypeOfAgentOffline
// 		} else {
// 			spans.Type = constant.TypeOfRegister
// 		}

// 		spanPackets.Payload = append(spanPackets.Payload, spans)
// 		payload, err := msgpack.Marshal(spanPackets)
// 		if err != nil {
// 			logger.Warn("msgpack Marshal", zap.String("error", err.Error()))
// 			continue
// 		}

// 		tracePacket := &network.TracePack{
// 			Type:       constant.TypeOfPinpoint,
// 			IsSync:     constant.TypeOfSyncNo,
// 			IsCompress: constant.TypeOfCompressNo,
// 			Payload:    payload,
// 		}

// 		if err := gAgent.collector.write(tracePacket); err != nil {
// 			logger.Warn("write info", zap.String("error", err.Error()))
// 			continue
// 		}

// 		time.Sleep(10 * time.Second)
// 	}
// }

// // reportAgentInfo 上报agent 信息
// func reportAgentInfo() {
// 	for {
// 		time.Sleep(1 * time.Second)
// 		if !gAgent.isLive {
// 			continue
// 		}
// 		break
// 	}
// 	for {
// 		time.Sleep(10 * time.Second)
// 		spanPackets := network.NewSpansPacket()
// 		spanPackets.Type = constant.TypeOfTCPData
// 		spanPackets.AppName = gAgent.appName
// 		spanPackets.AgentID = gAgent.agentID

// 		agentInfo, err := msgpack.Marshal(gAgent.agentInfo)
// 		if err != nil {
// 			logger.Warn("msgpack Marshal", zap.String("error", err.Error()))
// 			continue
// 		}
// 		spans := &network.Spans{
// 			Spans: agentInfo,
// 		}
// 		if gAgent.isLive == false {
// 			spans.Type = constant.TypeOfAgentOffline
// 		} else {
// 			spans.Type = constant.TypeOfRegister
// 		}

// 		spanPackets.Payload = append(spanPackets.Payload, spans)
// 		payload, err := msgpack.Marshal(spanPackets)
// 		if err != nil {
// 			logger.Warn("msgpack Marshal", zap.String("error", err.Error()))
// 			continue
// 		}

// 		id := gAgent.getSyncID()
// 		tracePacket := &network.TracePack{
// 			Type:       constant.TypeOfPinpoint,
// 			IsSync:     constant.TypeOfSyncYes,
// 			IsCompress: constant.TypeOfCompressNo,
// 			ID:         id,
// 			Payload:    payload,
// 		}

// 		if err := gAgent.collector.write(tracePacket); err != nil {
// 			logger.Warn("write info", zap.String("error", err.Error()))
// 			continue
// 		}

// 		// 创建chan
// 		if _, ok := gAgent.syncCall.newChan(id, 10); !ok {
// 			logger.Warn("syncCall newChan", zap.String("error", "创建sync chan失败"))
// 			continue
// 		}

// 		// 阻塞同步等待，并关闭chan
// 		if _, err := gAgent.syncCall.syncRead(id, 10, true); err != nil {
// 			logger.Warn("syncRead", zap.String("error", err.Error()))
// 			continue
// 		}
// 		time.Sleep(10 * time.Second)
// 	}
// }

// getSyncID ...
func (a *Agent) getSyncID() uint32 {
	return atomic.AddUint32(&a.syncID, 1)
}
