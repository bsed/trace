package service

import (
	"bufio"
	"io"
	"net"
	"time"

	"github.com/bsed/trace/agent/misc"
	"github.com/bsed/trace/pkg/constant"
	"github.com/bsed/trace/pkg/network"
	"github.com/vmihailenco/msgpack"
	"go.uber.org/zap"
)

// tcpClient tcp客户端， 用来和采集器通信
type tcpClient struct {
	isStart bool      // 是否启用
	conn    net.Conn  // 链接conn
	addr    string    // collector 地址
	quitC   chan bool // 退出信号
}

func newtcpClient(addr string) *tcpClient {
	return &tcpClient{
		isStart: false,
		addr:    addr,
		quitC:   make(chan bool, 1),
	}
}

// init 初始化链接
func (t *tcpClient) init() error {
	var err error
	quitC := make(chan bool, 1)
	// 定时器
	ticker := time.NewTicker(time.Duration(misc.Conf.Collector.Keeplive) * time.Second)

	defer func() {
		t.isStart = false
		if err := recover(); err != nil {
			logger.Warn("tcp init", zap.Stack("server"), zap.Any("err", err))
		}
	}()

	defer func() {
		close(quitC)
		if t.conn != nil {
			t.conn.Close()
		}
		ticker.Stop()
	}()

	t.conn, err = net.Dial("tcp", t.addr)
	if err != nil {
		logger.Warn("tcp connect", zap.String("err", err.Error()), zap.String("addr", t.addr))
		return err
	}

	t.isStart = true

	// 启动心跳
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := t.keeplive(); err != nil {
					logger.Warn("keeplive", zap.String("error", err.Error()))
					return
				}
				break
			case <-quitC:
				return
			}
		}
	}()
	reader := bufio.NewReaderSize(t.conn, constant.MaxMessageSize)
	for {
		select {
		case <-quitC:
			return nil
		default:
			packet, err := t.read(reader)
			if err != nil {
				logger.Warn("read", zap.Error(err))
				return err
			}
			// 发给上层处理
			switch packet.IsSync {
			case constant.TypeOfSyncYes:
				if err := gAgent.syncCall.syncWrite(packet.ID, packet); err != nil {
					logger.Warn("syncWrite", zap.Error(err))
				}
				break
			default:
				// gAgent.downloadC <- packet
				break
			}
		}
	}
}

// close 关闭链接
func (t *tcpClient) close() error {
	t.isStart = false
	if t.conn != nil {
		t.conn.Close()
	}
	return nil
}

// keeplive 心跳
func (t *tcpClient) keeplive() error {
	ping := network.NewPing()
	b, err := msgpack.Marshal(ping)
	if err != nil {
		logger.Warn("msgpack Marshal", zap.String("error", err.Error()))
		return err
	}

	cmd := network.NewCMD()
	cmd.Type = constant.TypeOfPing
	cmd.Payload = b

	buf, err := msgpack.Marshal(cmd)
	if err != nil {
		logger.Warn("msgpack Marshal", zap.String("error", err.Error()))
		return err
	}

	packet := &network.TracePack{
		Type:       constant.TypeOfCmd,
		IsSync:     constant.TypeOfSyncNo,
		IsCompress: constant.TypeOfCompressNo,
		ID:         0,
		Payload:    buf,
	}

	if err := t.write(packet); err != nil {
		logger.Warn("write", zap.String("error", err.Error()))
		return err
	}

	return nil
}

// read tcp读包
func (t *tcpClient) read(reader io.Reader) (*network.TracePack, error) {
	packet := &network.TracePack{}
	if err := packet.Decode(reader); err != nil {
		logger.Warn("tcp read decode", zap.String("error", err.Error()))
		return nil, err
	}
	return packet, nil
}

// write tcp写包
func (t *tcpClient) write(packet *network.TracePack) error {
	body := packet.Encode()
	if t.conn != nil {
		_, err := t.conn.Write(body)
		if err != nil {
			logger.Warn("tcp write", zap.String("error", err.Error()))
			return err
		}
	}
	return nil
}
