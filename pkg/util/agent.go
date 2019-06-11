package util

// Agent ...
type Agent struct {
	AppName      string `db:"app_name" json:"applicationName"  msg:"applicationName"`
	AgentID      string `db:"agent_id" json:"agentId"  msg:"agentId"`
	Type         int32  `db:"ser_type" json:"serviceType"  msg:"serviceType"`
	HostName     string `db:"host_name" json:"hostName" msg:"hostName"`
	IP4S         string `db:"ip" json:"ip" msg:"ip"`
	Pid          int32  `db:"pid" json:"pid" msg:"pid"`
	Version      string `db:"version" json:"version" msg:"version"`
	StartTime    int64  `db:"start_time" json:"startTimestamp" msg:"startTimestamp"`
	StopTime     int64  `db:"end_time" json:"end_time" msg:"end_time"`
	IsLive       bool   `db:"is_live" json:"is_live" msg:"is_live"`
	IsContainer  bool   `db:"is_container" json:"is_container" msg:"is_container"`
	OperatingEnv int32  `db:"operating_env" json:"operating_env" msg:"operating_env"`
}

// NewAgent ...
func NewAgent() *Agent {
	return &Agent{}
}
