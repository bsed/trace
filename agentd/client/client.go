package client

import (
	"bytes"
	"fmt"
	"math/rand"
	"os/exec"
	"time"

	"github.com/imdevlab/g/utils"

	"go.uber.org/zap"

	"github.com/imdevlab/g"
	"github.com/bsed/trace/agentd/misc"
	"github.com/bsed/trace/pkg/util"
	"github.com/valyala/fasthttp"
)

var c = &fasthttp.Client{}

type Client struct {
	StartTime        int64
	DownloadTimes    int
	RestartTimes     int
	LastDownloadTime int64
	LastRestartTime  int64
	Hostname         string
	AdminPort        string
}

func New() *Client {
	hn, err := utils.Hostname()
	if err != nil {
		g.L.Fatal("can't get the hostname", zap.Error(err))
	}

	return &Client{
		StartTime: time.Now().Unix(),
		Hostname:  hn,
		AdminPort: misc.Conf.Client.AdminPort,
	}
}

var downloadVersion string

func (cli *Client) Start() {
	go cli.initAdmin()

	// 获取当前最新的agent版本号
	downloadVersion, _ = util.GetVersion("version")
	g.L.Info("local agent version:", zap.String("version", downloadVersion))

	// 本地已经下载过agent，先启动旧的agent
	if downloadVersion != "" {
		cli.agentStopStart()
	}

	// 随机等待1-60秒，避免因为同时大量的更新导致对中心服务器的冲击
	r := rand.Int63n(60) + 1
	time.Sleep(time.Duration(r) * time.Second)

	// 定时上报自身信息，同时获取最新的版本号
	url := "http://" + misc.Conf.Client.ServerAddr + "/web/agentd/update"

	go func() {
		for {
			newVer := cli.update(url)
			if newVer == "" {
				goto SLEEP
			}
			if newVer != downloadVersion {
				g.L.Info("find new agent version,start to download", zap.String("version", newVer))
				// 发现新版本,下载
				err := cli.download()
				if err != nil {
					// download任何一个环节出问题，都不要更新当前版本，继续下载
					// 只有版本号成功更新之时，才是download成功之时
					g.L.Warn("download error", zap.Error(err))
					goto SLEEP
				}

				err = cli.agentStopStart()
				if err != nil {
					goto SLEEP
				}
				// 下载并重启成功，更新version
				downloadVersion = newVer
			} else {
				// 监督服务是否挂了，如果挂了，需要拉起来
				if !isAlive() {
					g.L.Warn("agent no alive, restaring...")
					cli.startAgent()
				}
			}

		SLEEP:
			// update interval: 120 - 180s
			r := rand.Int63n(60) + 120
			time.Sleep(time.Duration(r) * time.Second)
		}
	}()
}

func (cli *Client) agentStopStart() error {
	stopAgent()
	time.Sleep(1 * time.Second)
	return cli.startAgent()
}
func (cli *Client) download() error {
	cli.DownloadTimes++
	cli.LastDownloadTime = time.Now().Unix()

	// 删除旧的zip包
	delZip()

	url := "http://" + misc.Conf.Client.ServerAddr + "/web/agentd/download"
	// 下载agent.zip
	nurl := url + "/agent.zip"

	in := bytes.NewBuffer(nil)
	cmd := exec.Command("/bin/bash")
	cmd.Stdin = in
	in.WriteString("wget -O agent.zip " + nurl)
	bs, err := cmd.CombinedOutput()
	if err != nil {
		g.L.Warn("download agent.zip error", zap.Error(err), zap.String("url", nurl), zap.String("res", string(bs)))
		return err
	}

	time.Sleep(2 * time.Second)

	// 下载成功后，先关闭agent服务，删除本地的文件夹，然后解压

	// 解压agent.zip
	in = bytes.NewBuffer(nil)
	cmd = exec.Command("/bin/bash")
	cmd.Stdin = in

	s := fmt.Sprintf("unzip -o agent.zip")
	in.WriteString(s)
	bs, err = cmd.CombinedOutput()
	if err != nil {
		g.L.Warn("unzip agent.zip error", zap.Error(err), zap.String("res", string(bs)))
		delZip()
		return err
	}

	// 下载version
	nurl = url + "/version"
	in = bytes.NewBuffer(nil)
	cmd = exec.Command("/bin/bash")
	cmd.Stdin = in
	in.WriteString("wget -O version " + nurl)
	bs, err = cmd.CombinedOutput()
	if err != nil {
		g.L.Warn("download version error", zap.Error(err), zap.String("url", nurl), zap.String("res", string(bs)))
		return err
	}

	return nil
}

func stopAgent() error {
	c := fmt.Sprintf("pkill -9 apm-agent")
	cmd := exec.Command("/bin/sh", "-c", c)
	cmd.CombinedOutput()

	return nil
}

func (cli *Client) startAgent() error {
	cli.RestartTimes++
	cli.LastRestartTime = time.Now().Unix()

	g.L.Info("agent starting...")
	// 进入apm-agent目录
	c := fmt.Sprintf("`nohup ./apm-agent 1>out.log 2>error.log &` && echo '启动成功' && exit")
	cmd := exec.Command("/bin/sh", "-c", c)
	cmd.Dir = "./release/apm-agent"

	bs, err := cmd.CombinedOutput()
	if err != nil {
		g.L.Warn("start agent error", zap.Error(err), zap.String("res", string(bs)))
		return err
	}

	time.Sleep(5 * time.Second)
	if !isAlive() {
		g.L.Warn("agent start failed, please see detail logs in release/apm-agent/error.log")
		return fmt.Errorf("agent not alive after start")
	}
	g.L.Info("agent started ok")

	return nil
}

func isAlive() bool {
	c := fmt.Sprintf("ps -ef | grep %s | grep -v 'grep ' | awk '{print $2}'", "apm-agent")
	cmd := exec.Command("/bin/sh", "-c", c)

	bs, err := cmd.CombinedOutput()
	if err != nil || string(bs) == "" {
		return false
	}

	return true
}

func delZip() {
	c := fmt.Sprintf("rm -f agent.zip && rm -f version")
	cmd := exec.Command("/bin/sh", "-c", c)

	bs, err := cmd.CombinedOutput()
	if err != nil {
		g.L.Warn("del agent.zip error", zap.Error(err), zap.String("res", string(bs)))
	}
}
