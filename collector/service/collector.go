package service

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/imdevlab/g"

	"github.com/bsed/trace/pkg/alert"

	"github.com/bsed/trace/pkg/mq"

	"go.uber.org/zap"

	"github.com/bsed/trace/collector/misc"
	"github.com/bsed/trace/collector/storage"
	"github.com/bsed/trace/collector/ticker"
	"github.com/bsed/trace/pkg/constant"
	"github.com/bsed/trace/pkg/network"
	"github.com/vmihailenco/msgpack"
)

var logger *zap.Logger

// Collector 采集服务
type Collector struct {
	sync.RWMutex
	etcd       *Etcd               // 服务上报
	apps       *Apps               // app集合
	ticker     *ticker.Tickers     // 定时器
	apiTicker  *ticker.Tickers     // 定时器
	storage    *storage.Storage    // 存储
	mq         *mq.Nats            // 消息队列
	pushC      chan *alert.Data    // 推送通道
	collectors map[string]struct{} // collectors
	hash       *g.Hash             // 一致性hash
}

var gCollector *Collector

// New new collecotr
func New(l *zap.Logger) *Collector {
	logger = l
	gCollector = &Collector{
		etcd:       newEtcd(),
		apps:       newApps(),
		ticker:     ticker.NewTickers(misc.Conf.Ticker.Num, misc.Conf.Ticker.Interval, logger),
		apiTicker:  ticker.NewTickers(misc.Conf.Ticker.Num, misc.Conf.Apps.ApiStatsInterval, logger),
		storage:    storage.NewStorage(logger),
		mq:         mq.NewNats(logger),
		pushC:      make(chan *alert.Data, 3000),
		collectors: make(map[string]struct{}), // collectors
		hash:       g.NewHash(),
	}
	return gCollector
}

// Start 启动collector
func (c *Collector) Start() error {
	// 启动mq服务
	if err := c.mq.Start(misc.Conf.MQ.Addrs); err != nil {
		logger.Warn("mq start  error", zap.String("error", err.Error()))
		return err
	}

	// 启动存储服务
	if err := c.storage.Start(); err != nil {
		logger.Warn("storage start  error", zap.String("error", err.Error()))
		return err
	}

	// 存储服务类型
	if err := c.apps.start(); err != nil {
		logger.Warn("apps start error", zap.String("error", err.Error()))
		return err
	}

	// 初始化上报key
	key, err := reportKey(misc.Conf.Etcd.ReportDir)
	if err != nil {
		logger.Warn("get reportKey error", zap.String("error", err.Error()))
		return err
	}

	// 初始化etcd
	if err := c.etcd.Init(misc.Conf.Etcd.Addrs, key, misc.Conf.Collector.Addr); err != nil {
		logger.Warn("etcd init error", zap.String("error", err.Error()))
		return err
	}

	// 启动etcd服务
	if err := c.etcd.Start(); err != nil {
		logger.Warn("etcd start error", zap.String("error", err.Error()))
		return err
	}

	// 订阅
	if err := c.mq.Subscribe(c.etcd.ReportKey, msgHandle); err != nil {
		logger.Warn("mq subscribe  error", zap.String("error", err.Error()))
		return err
	}

	// 启动tcp服务
	if err := c.startNetwork(); err != nil {
		logger.Warn("start network error", zap.String("error", err.Error()))
		return err
	}

	// 启动推送服务
	if err := c.pushWork(); err != nil {
		logger.Warn("start push work error", zap.String("error", err.Error()))
		return err
	}

	logger.Info("Collector start ok")
	return nil
}

// Close 关闭collector
func (c *Collector) Close() error {
	close(c.pushC)
	return nil
}

func reportKey(dir string) (string, error) {
	value, err := collectorName()
	if err != nil {
		return "", err
	}

	dirLen := len(dir)
	if dirLen > 0 && dir[dirLen-1] != '/' {
		return dir + "/" + value, nil
	}
	return dir + value, nil
}

// collectorName etcd 上报key
func collectorName() (string, error) {
	host, err := os.Hostname()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%d", host, os.Getpid()), nil
}

func initDir(dir string) string {
	dirLen := len(dir)
	if dirLen > 0 && dir[dirLen-1] != '/' {
		return dir + "/"
	}
	return dir
}

// cmdPacket 处理agent 发送来的cmd报文
func cmdPacket(conn net.Conn, packet *network.TracePack) error {
	cmd := network.NewCMD()
	if err := msgpack.Unmarshal(packet.Payload, cmd); err != nil {
		logger.Warn("msgpack Unmarshal", zap.String("error", err.Error()))
		return err
	}
	switch cmd.Type {
	case constant.TypeOfPing:
		ping := network.NewPing()
		if err := msgpack.Unmarshal(cmd.Payload, ping); err != nil {
			logger.Warn("msgpack Unmarshal", zap.String("error", err.Error()))
			return err
		}
		// logger.Debug("ping", zap.String("addr", conn.RemoteAddr().String()))
	}
	return nil
}

// start 启动tcp服务
func (c *Collector) startNetwork() error {
	lsocket, err := net.Listen("tcp", misc.Conf.Collector.Addr)
	if err != nil {
		logger.Fatal("Listen", zap.String("msg", err.Error()), zap.String("addr", misc.Conf.Collector.Addr))
	}

	go func() {
		for {
			conn, err := lsocket.Accept()
			if err != nil {
				logger.Fatal("Accept", zap.String("msg", err.Error()), zap.String("addr", misc.Conf.Collector.Addr))
			}
			conn.SetReadDeadline(time.Now().Add(time.Duration(misc.Conf.Collector.Timeout) * time.Second))
			tcpClient := newtcpClient()
			go tcpClient.start(conn)
		}
	}()
	return nil
}

type tcpClient struct {
	appName string
	agentID string
}

func newtcpClient() *tcpClient {
	return &tcpClient{}
}

func (t *tcpClient) start(conn net.Conn) {
	quitC := make(chan bool, 1)
	packetC := make(chan *network.TracePack, 100)

	defer func() {
		if err := gCollector.storage.UpdateAgentState(t.appName, t.agentID, false); err != nil {
			logger.Warn("tcp close , update agent state Store", zap.String("error", err.Error()))
		}
		if err := recover(); err != nil {
			logger.Error("tcpClient", zap.Any("msg", err))
			return
		}
	}()

	defer func() {
		if conn != nil {
			conn.Close()
		}
		close(quitC)
	}()

	go t.tcpRead(conn, packetC, quitC)

	for {
		select {
		case packet, ok := <-packetC:
			if !ok {
				logger.Info("quit")
				return
			}
			switch packet.Type {
			case constant.TypeOfCmd:
				if err := cmdPacket(conn, packet); err != nil {
					logger.Warn("cmd packet", zap.String("error", err.Error()))
					return
				}
				break
			case constant.TypeOfPinpoint:
				if err := t.pinpointPacket(conn, packet); err != nil {
					logger.Warn("pinpoint packet", zap.String("error", err.Error()))
					return
				}
				break
			case constant.TypeOfSystem:

				log.Println("TypeOfSystem")
				break
			}
		}
	}

}

func (t *tcpClient) tcpRead(conn net.Conn, packetC chan *network.TracePack, quitC chan bool) {
	defer func() {
		if err := recover(); err != nil {
			logger.Error("tcpRead recover", zap.Any("error", err))
			return
		}
	}()
	defer func() {
		close(packetC)
	}()
	reader := bufio.NewReaderSize(conn, constant.MaxMessageSize)
	for {
		select {
		case <-quitC:
			break
		default:
			packet := network.NewTracePack()
			if err := packet.Decode(reader); err != nil {
				logger.Warn("tcp read error", zap.String("err", err.Error()))
				return
			}
			packetC <- packet
			// 设置超时时间
			conn.SetReadDeadline(time.Now().Add(time.Duration(misc.Conf.Collector.Timeout) * time.Second))
		}
	}
}

func (c *Collector) pushWork() error {
	for {
		select {
		case packet, ok := <-c.pushC:
			if ok {
				data, err := msgpack.Marshal(packet)
				if err != nil {
					logger.Warn("msgpack", zap.String("error", err.Error()))
					break
				}
				if err := c.mq.Publish(misc.Conf.MQ.Topic, data); err != nil {
					logger.Warn("publish", zap.Error(err))
				}
			}
			break
		}
	}
	// return nil
}

func (c *Collector) publish(data *alert.Data) {
	c.pushC <- data
}

func (c *Collector) addCollector(key string) {
	c.RLock()
	_, ok := c.collectors[key]
	c.RUnlock()
	if !ok {
		c.Lock()
		c.collectors[key] = struct{}{}
		c.Unlock()
		c.hash.Add(key)
	}
}

func (c *Collector) removeCollector(key string) {
	c.RLock()
	_, ok := c.collectors[key]
	c.RUnlock()
	if ok {
		c.Lock()
		delete(c.collectors, key)
		c.Unlock()
		c.hash.Remove(key)
	}
}

// getCollecotorTopic 获取collector主题
func (c *Collector) getCollecotorTopic(appName string) (string, error) {
	topic, err := c.hash.Get(appName)
	if err != nil {
		return "", err
	}
	return topic, nil
}

func getblockIndex(value int) int {
	if 0 <= value && value <= 15 {
		return 0
	} else if 16 <= value && value <= 30 {
		return 1
	} else if 31 <= value && value <= 45 {
		return 2
	} else if 46 <= value && value <= 59 {
		return 3
	}
	return 0
}

// getNameByIP 通过ip获取应用名
func getNameByIP(ip string) (string, bool) {
	appName, ok := gCollector.apps.getNameByIP(ip)
	return appName, ok
}

// getNameByDubboAPI 通过api获取应用名
func getNameByDubboAPI(api string) (string, bool) {
	appName, ok := gCollector.apps.dubbo.Get(api)
	return appName, ok
}
