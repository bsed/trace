package service

import (
	"context"
	"time"

	"github.com/bsed/trace/collector/misc"
	"go.etcd.io/etcd/clientv3"

	"go.uber.org/zap"
)

// Etcd struct,report udp listen addr
type Etcd struct {
	Client      *clientv3.Client
	Grant       *clientv3.LeaseGrantResponse
	ReportKey   string
	ReportValue string
	StopC       chan bool
}

// newEtcd new Etcd
func newEtcd() *Etcd {
	etcd := &Etcd{
		StopC: make(chan bool, 1),
	}
	return etcd
}

// Start start report thread
func (e *Etcd) Start() error {
	go e.registerWork()
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
func (e *Etcd) Init(addrs []string, reportKey, reportValue string) error {
	e.ReportKey = reportKey
	e.ReportValue = reportValue

	logger.Info("Init", zap.String("@Key", e.ReportKey), zap.String("@Value", e.ReportValue))

	cfg := clientv3.Config{
		Endpoints:   addrs,
		DialTimeout: time.Duration(misc.Conf.Etcd.TimeOut) * time.Second,
	}

	client, err := clientv3.New(cfg)
	if err != nil {
		return err
	}

	e.Client = client

	return nil
}

// registerWork register stream addr
func (e *Etcd) registerWork() {
	timeC := time.Tick(time.Duration(misc.Conf.Etcd.ReportTime) * time.Second)
	// 启动立刻注册一次
	e.Put(e.ReportKey, e.ReportValue, misc.Conf.Etcd.TTL)
	for {
		select {
		case <-e.StopC:
			logger.Info("Etcd", zap.String("Close", "Ok"))
			return
		// Timing task
		case <-timeC:
			if err := e.Put(e.ReportKey, e.ReportValue, misc.Conf.Etcd.TTL); err != nil {
				logger.Warn("Etcd", zap.String("error", err.Error()))
			}
			break
		}
	}
}

// Put put
func (e *Etcd) Put(key, value string, ttl int64) error {
	Grant, err := e.Client.Grant(context.TODO(), ttl)
	if err != nil {
		logger.Error("Etcd", zap.Error(err), zap.Int64("@ReportTime", ttl))
		return err
	}
	_, err = e.Client.Put(context.TODO(), key, value, clientv3.WithLease(Grant.ID))
	if err != nil {
		logger.Error("Put", zap.String("@key", key), zap.String("@value", value), zap.Error(err))
		return err
	}
	return nil
}

// Get Get
func (e *Etcd) Get() {
	for {
		rch := e.Client.Watch(context.Background(), misc.Conf.Etcd.ReportDir, clientv3.WithPrefix())
		for wresp := range rch {
			for _, ev := range wresp.Events {
				// 上报
				if ev.Type == 0 {
					gCollector.addCollector(string(ev.Kv.Key))
				} else {
					gCollector.removeCollector(string(ev.Kv.Key))
				}
			}
		}
	}
}
