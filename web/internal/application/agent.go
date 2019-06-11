package app

import (
	"encoding/json"

	"github.com/imdevlab/g"
	"github.com/imdevlab/g/utils"
	"github.com/bsed/trace/web/internal/misc"
	"go.uber.org/zap"
)

type Agent struct {
	AgentID  string `json:"agent_id"`
	HostName string `json:"host_name"`
	IP       string `json:"ip"`

	IsLive      bool `json:"is_live"`
	IsContainer bool `json:"is_container"`

	StartTime    string     `json:"start_time"`
	SocketID     int        `json:"socket_id"`
	OperatingEnv int        `json:"operating_env"`
	TracingAddr  string     `json:"tracing_addr "`
	Info         *AgentInfo `json:"info"`
}

type AgentInfo struct {
	AgentVersion   string          `json:"agentVersion"`
	VmVersion      string          `json:"vmVersion"`
	Pid            int             `json:"pid"`
	ServerMetaData *ServerMetaData `json:"serverMetaData"`
}

type ServerMetaData struct {
	ServerInfo string   `json:"serverInfo"`
	VmArgs     []string `json:"vmArgs"`
}

func queryAgents(app string) ([]*Agent, error) {
	q := misc.StaticCql.Query(`SELECT agent_id,host_name,ip,is_live,is_container,start_time,socket_id,operating_env,tracing_addr,agent_info FROM agents WHERE app_name=?`, app)
	iter := q.Iter()

	var agentID, hostName, ip, tracingAddr, info string
	var isLive, isContainer bool
	var socketID, operatingEnv int
	var startTime int64

	agents := make([]*Agent, 0)
	for iter.Scan(&agentID, &hostName, &ip, &isLive, &isContainer, &startTime, &socketID, &operatingEnv, &tracingAddr, &info) {
		agent := &Agent{
			AgentID:      agentID,
			HostName:     hostName,
			IP:           ip,
			IsLive:       isLive,
			IsContainer:  isContainer,
			StartTime:    utils.UnixMsToTimestring(startTime),
			SocketID:     socketID,
			OperatingEnv: operatingEnv,
			TracingAddr:  tracingAddr,
		}
		ai := &AgentInfo{}
		json.Unmarshal([]byte(info), &ai)
		agent.Info = ai

		agents = append(agents, agent)
	}

	if err := iter.Close(); err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
		return nil, err
	}

	return agents, nil
}

func countAgentsAlive() (map[string]int, map[string]int) {
	q := misc.StaticCql.Query(`SELECT app_name,is_live FROM agents`)
	iter := q.Iter()

	var appName string
	var isLive bool

	alive := make(map[string]int)
	unalive := make(map[string]int)
	for iter.Scan(&appName, &isLive) {
		if isLive {
			count := alive[appName]
			alive[appName] = count + 1
		} else {
			count := unalive[appName]
			unalive[appName] = count + 1
		}
	}

	if err := iter.Close(); err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
		return nil, nil
	}

	return alive, unalive
}
