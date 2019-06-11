package client

import (
	"encoding/json"
	"time"

	"github.com/imdevlab/g"
	"github.com/bsed/trace/agentd/misc"
	"github.com/bsed/trace/pkg/util"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

func (cli *Client) update(url string) string {
	// 获取本地正在运行的agent的health和version
	ag := checkAgentHealth()

	// 上报当前的状态到agentd-server
	var arg fasthttp.Args
	info := &util.AgentdInfo{cli.Hostname, cli.StartTime, cli.DownloadTimes, cli.RestartTimes, cli.LastDownloadTime, cli.LastRestartTime,
		misc.Conf.Common.Version, ag, downloadVersion, cli.AdminPort, time.Now().Unix()}
	b, _ := json.Marshal(info)
	arg.SetBytesV("info", b)
	code, r, err := c.Post(nil, url, &arg)

	if code != 200 || err != nil {
		g.L.Warn("update to agentd-server error", zap.Error(err), zap.Int("code", code))
		return ""
	}

	// 返回最新的版本号
	return string(r)
}

// 获取本地运行的agent的信息
func checkAgentHealth() string {
	var arg fasthttp.Args
	code, r, err := c.Post(nil, misc.Conf.Client.AgentHealthURL, &arg)
	if code != 200 || err != nil {
		g.L.Warn("check agent  health error", zap.Error(err), zap.Int("code", code))
		return ""
	}

	hr := util.HealthResult{}
	json.Unmarshal(r, &hr)
	return hr.Version
}
