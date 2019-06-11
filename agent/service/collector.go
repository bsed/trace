package service

import (
	"fmt"
	"strings"
	"sync"

	"github.com/bsed/trace/pkg/network"

	"go.uber.org/zap"

	"github.com/imdevlab/g"
)

// Collector 收集器
type Collector struct {
	hash *g.Hash
	sync.RWMutex
	key     string                // 使用的key
	clients map[string]*tcpClient // kehuduan
}

func (c *Collector) add(key, addr string) error {
	c.RLock()
	newClient, ok := c.clients[key]
	c.RUnlock()
	if !ok {
		newClient = newtcpClient(addr)
		c.Lock()
		c.clients[key] = newClient
		c.Unlock()
		// 添加到hash
		c.hash.Add(key)
		ruleKey, err := c.hash.Get(gAgent.appName)
		if err != nil {
			logger.Warn("hash get", zap.String("error", err.Error()))
			return err
		}
		// 链接
		if !strings.EqualFold(ruleKey, c.key) {
			// 查看其他链接，如果有链接就关闭
			c.RLock()
			for _, oldClient := range c.clients {
				oldClient.close()
			}
			c.RUnlock()
			// 新链接启动
			go newClient.init()
			logger.Info("new Conn", zap.String("addr", newClient.addr), zap.String("key", ruleKey))
			// 保存key
			c.key = ruleKey
		}
		return nil
	}

	// 已经存在的上报的信息也需要检查
	ruleKey, err := c.hash.Get(gAgent.appName)
	if err != nil {
		logger.Warn("hash get", zap.String("error", err.Error()))
		return err
	}

	if !strings.EqualFold(ruleKey, c.key) {
		// 存在并符合链接，检查是否已经建立链接
		c.RLock()
		for _, oldClient := range c.clients {
			oldClient.close()
		}

		c.RUnlock()
		go newClient.init()
		logger.Info("new Conn", zap.String("addr", newClient.addr), zap.String("key", ruleKey))
		// 保存key
		c.key = ruleKey
	} else {
		// 重连
		if !newClient.isStart {
			go newClient.init()
			logger.Info("client reConn", zap.String("addr", newClient.addr), zap.String("key", ruleKey))
		}
	}

	return nil
}

func (c *Collector) del(key string) error {

	c.RLock()
	client, ok := c.clients[key]
	c.RUnlock()

	if ok {
		c.hash.Remove(key)
		c.Lock()
		delete(c.clients, key)
		c.Unlock()
		client.close()
	}
	return nil
}

func newCollector() *Collector {
	return &Collector{
		hash:    g.NewHash(),
		clients: make(map[string]*tcpClient),
	}
}

// write write.
func (c *Collector) write(packet *network.TracePack) error {
	key, err := c.hash.Get(gAgent.appName)
	if err != nil {
		logger.Warn("write", zap.String("error", err.Error()))
		return err
	}
	// 可优化，去除锁
	c.RLock()
	client, ok := c.clients[key]
	c.RUnlock()
	if !ok {
		return fmt.Errorf("no server, key is %s", key)
	}

	// 发送
	if err = client.write(packet); err != nil {
		logger.Warn("write", zap.String("error", err.Error()))
		return err
	}
	return nil
}
