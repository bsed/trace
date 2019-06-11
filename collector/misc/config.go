package misc

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

// Config ...
type Config struct {
	Common struct {
		Version    string
		LogLevel   string
		AdminToken string
	}

	Collector struct {
		Addr    string
		Timeout int
	}

	Etcd struct {
		Addrs      []string
		TimeOut    int
		ReportTime int
		ReportDir  string
		TTL        int64
	}

	Storage struct {
		Cluster             []string
		TraceKeyspace       string
		StaticKeyspace      string
		NumConns            int
		SpanCacheLen        int // 缓存长度
		SpanChunkCacheLen   int // 缓存长度
		MetricCacheLen      int // 缓存长度
		SpanStoreInterval   int // 毫秒
		SystemStoreInterval int // 毫秒
		AgentStatUseTTL     bool
		AgentStatTTL        int64
		GoruntineNum        int // 入库goruntine数量
	}

	Stats struct {
		DeferTime        int64   // 延迟计算时间，单位秒
		MapRange         int64   // 应用拓扑图计算时间范围，单位秒
		APICallRange     int64   // API应用调用计算时间范围，单位秒
		SatisfactionTime int32   // APDEX 满意时间指标，单位毫秒
		TolerateTime     int32   // APDEX 可容忍时间指标，单位毫秒
		RuntimeRange     int64   // Runtime延迟计算时间
		DefaultCode      []int32 // http默认code
	}

	Apps struct {
		LoadInterval     int64 // 加载app时间间隔
		ApiStatsInterval int64 // api二次聚合延迟时间
	}

	MQ struct {
		Addrs []string // mq地址
		Topic string   // 主题
	}

	Ticker struct {
		Num      int   // 定时器个数
		Interval int64 // 任务时间间隔
	}
}

// Conf ...
var Conf *Config

// InitConfig ...
func InitConfig(path string) {
	conf := &Config{}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal("InitConfig:ioutil.ReadFile", err)
	}

	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		log.Fatal("InitConfig:yaml.Unmarshal", err)
	}
	Conf = conf
	log.Println(Conf)
}
