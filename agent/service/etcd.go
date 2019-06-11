package service

import (
	"context"
	"time"

	"github.com/bsed/trace/agent/misc"

	"go.etcd.io/etcd/clientv3"

	"go.uber.org/zap"
)

// Etcd struct,report udp listen addr
type Etcd struct {
	Client *clientv3.Client
	Grant  *clientv3.LeaseGrantResponse
	StopC  chan bool
}

// newEtcd new Etcd
func newEtcd() *Etcd {
	etcd := &Etcd{
		StopC: make(chan bool, 1),
	}
	return etcd
}

// Start start ereport thread
func (e *Etcd) Start() error {
	// go e.registerWork()
	go e.Get()
	return nil
}

// Close stop ereport
func (e *Etcd) Close() {
	e.StopC <- true
	time.Sleep(1 * time.Second)

	// close channel
	close(e.StopC)

	if e.Client != nil {
		e.Client.Close()
	}
}

// Init init Etcd
func (e *Etcd) Init() error {
	cfg := clientv3.Config{
		Endpoints:   misc.Conf.Etcd.Addrs,
		DialTimeout: time.Duration(misc.Conf.Etcd.TimeOut) * time.Second,
	}

	client, err := clientv3.New(cfg)
	if err != nil {
		return err
	}

	e.Client = client

	//检查etcd连接是否成功
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(misc.Conf.Etcd.TimeOut)*time.Second)
	_, err = e.Client.Cluster.MemberList(ctx)
	if err != nil {
		return err
	}
	return nil
}

// Put put
func (e *Etcd) Put(key, value string, ttl int64) error {
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(misc.Conf.Etcd.TimeOut)*time.Second)
	Grant, err := e.Client.Grant(ctx, ttl)
	if err != nil {
		logger.Error("Etcd", zap.Error(err), zap.Int64("@ReportTime", ttl))
		return err
	}
	_, err = e.Client.Put(ctx, key, value, clientv3.WithLease(Grant.ID))
	if err != nil {
		logger.Error("Put", zap.String("@key", key), zap.String("@value", value), zap.Error(err))
		return err
	}
	return nil
}

// Get Get
func (e *Etcd) Get() {
	for {
		rch := e.Client.Watch(context.Background(), misc.Conf.Etcd.WatchDir, clientv3.WithPrefix())
		for wresp := range rch {
			for _, ev := range wresp.Events {
				// 上报
				if ev.Type == 0 {
					gAgent.collector.add(string(ev.Kv.Key), string(ev.Kv.Value))
				} else {
					gAgent.collector.del(string(ev.Kv.Key))
				}
			}
		}
	}
}
