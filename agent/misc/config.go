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

	Agent struct {
	}

	Collector struct {
		Keeplive int
	}

	Etcd struct {
		Addrs    []string
		WatchDir string
		TimeOut  int
	}

	Pinpoint struct {
		InfoAddr           string // tcp addr for info
		StatAddr           string // udp addr for stat
		SpanAddr           string // udp addr for span
		SpanReportInterval int    // 全链路信息上报频率 单位毫秒
		SpanQueueLen       int
	}

	Health struct {
		Addr string
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
