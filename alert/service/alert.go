package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/gocql/gocql"
	"github.com/bsed/trace/alert/control"
	"github.com/bsed/trace/alert/misc"
	"github.com/bsed/trace/alert/ticker"
	"github.com/bsed/trace/pkg/mq"
	"github.com/bsed/trace/pkg/sql"
	"go.uber.org/zap"
)

var logger *zap.Logger

// Alert 告警服务
type Alert struct {
	mutex     sync.Mutex
	apps      *Apps            // app集合
	staticCql *gocql.Session   // 静态数据客户端
	traceCql  *gocql.Session   // 动态数据客户端
	tickers   *ticker.Tickers  // 定时任务
	mq        *mq.Nats         // 消息队列
	control   *control.Control // 告警控制中心
	alertID   int64            // 告警ID
}

var gAlert *Alert

// New new alert
func New(l *zap.Logger) *Alert {
	logger = l
	gAlert = &Alert{
		apps:    newApps(),
		tickers: ticker.NewTickers(10, misc.Conf.Ticker.Interval, logger),
		mq:      mq.NewNats(logger),
		control: control.New(&misc.Conf.Control, logger),
		alertID: time.Now().Unix() * 1000,
	}
	return gAlert
}

// Start start server
func (a *Alert) Start() error {
	// 初始化存储
	if err := a.initDB(); err != nil {
		logger.Warn("int db", zap.String("error", err.Error()))
		return err
	}
	// 初始化控制中心
	if err := a.control.Init(gettraceCql); err != nil {
		logger.Warn("int control", zap.String("error", err.Error()))
		return err
	}
	// 加载apps
	if err := a.apps.start(); err != nil {
		logger.Warn("apps start", zap.String("error", err.Error()))
		return err
	}
	// 启动mq服务
	if err := a.mq.Start(misc.Conf.MQ.Addrs); err != nil {
		logger.Warn("mq start  error", zap.String("error", err.Error()))
		return err
	}
	// 加载用户信息
	if err := a.loadUserSrv(); err != nil {
		logger.Warn("load users", zap.String("error", err.Error()))
		return err
	}
	// 订阅
	if err := a.mq.Subscribe(misc.Conf.MQ.Topic, msgHandle); err != nil {
		logger.Warn("mq subscribe  error", zap.String("error", err.Error()))
		return err
	}

	return nil
}

// Close stop server
func (a *Alert) Close() error {
	return nil
}

// initDB 初始化存储
func (a *Alert) initDB() error {
	if err := a.initTraceCql(); err != nil {
		logger.Warn("init trace cql error", zap.String("error", err.Error()))
		return err
	}

	if err := a.initStaticCql(); err != nil {
		logger.Warn("init static cql error", zap.String("error", err.Error()))
		return err
	}

	return nil
}

func (a *Alert) initTraceCql() error {
	// connect to the cluster
	cluster := gocql.NewCluster(misc.Conf.DB.Cluster...)
	cluster.Keyspace = misc.Conf.DB.TraceKeyspace
	cluster.Consistency = gocql.Quorum
	//设置连接池的数量,默认是2个（针对每一个host,都建立起NumConns个连接）
	cluster.NumConns = misc.Conf.DB.NumConns
	cluster.ReconnectInterval = 1 * time.Second
	session, err := cluster.CreateSession()
	if err != nil {
		logger.Warn("create session", zap.String("error", err.Error()))
		return err
	}
	a.traceCql = session
	return nil
}

func (a *Alert) initStaticCql() error {
	// connect to the cluster
	cluster := gocql.NewCluster(misc.Conf.DB.Cluster...)
	cluster.Keyspace = misc.Conf.DB.StaticKeyspace
	cluster.Consistency = gocql.Quorum
	//设置连接池的数量,默认是2个（针对每一个host,都建立起NumConns个连接）
	cluster.NumConns = misc.Conf.DB.NumConns
	cluster.ReconnectInterval = 1 * time.Second
	session, err := cluster.CreateSession()
	if err != nil {
		logger.Warn("create session", zap.String("error", err.Error()))
		return err
	}
	a.staticCql = session
	return nil
}

// GetCql ...
func (a *Alert) GettraceCql() *gocql.Session {
	if a.traceCql != nil {
		return a.traceCql
	}
	return nil
}

func (a *Alert) GetStaticCql() *gocql.Session {
	if a.staticCql != nil {
		return a.staticCql
	}
	return nil
}

func gettraceCql() *gocql.Session {
	return gAlert.GettraceCql()
}

func (a *Alert) loadUserSrv() error {
	if err := a.loadUser(); err != nil {
		logger.Warn("load user error", zap.String("error", err.Error()))
		return err
	}

	go func() {
		for {
			time.Sleep(time.Duration(misc.Conf.App.LoadInterval) * time.Second)
			if err := a.loadUser(); err != nil {
				logger.Warn("load user error", zap.String("error", err.Error()))
			}
		}
	}()

	return nil
}

func (a *Alert) loadUser() error {
	cql := gAlert.GetStaticCql()
	if cql == nil {
		return fmt.Errorf("unfind cql")
	}

	query := cql.Query(sql.LoadUers).Iter()
	defer func() {
		if err := query.Close(); err != nil {
			logger.Warn("close iter error:", zap.Error(err))
		}
	}()
	var id, email, mobile string
	for query.Scan(&id, &email, &mobile) {
		a.control.AddUser(id, email, mobile)
	}

	return nil
}

func (a *Alert) getAlertID() int64 {
	a.mutex.Lock()
	a.alertID++
	a.mutex.Unlock()
	return a.alertID
}
