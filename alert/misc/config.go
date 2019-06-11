package misc

import (
	"io/ioutil"
	"log"

	"github.com/bsed/trace/alert/control"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Common struct {
		Version    string
		LogLevel   string
		AdminToken string
	}
	MQ struct {
		Addrs []string // mq地址
		Topic string   // 主题
	}

	App struct {
		LoadInterval int
	}

	DB struct {
		Cluster        []string
		TraceKeyspace  string
		StaticKeyspace string
		NumConns       int
	}

	Ticker struct {
		Interval int
	}

	Control control.Conf
}

var Conf *Config

func InitConfig(path string) {
	conf := &Config{}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal("read config error :", err)
	}

	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		log.Fatal("yaml decode error :", err)
	}
	Conf = conf
}
