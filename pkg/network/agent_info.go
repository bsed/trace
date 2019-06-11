package network

// AgentInfo ...
type AgentInfo struct {
	AppName        string `db:"app_name" json:"applicationName"  msg:"applicationName"`
	AgentID        string `db:"agent_id" json:"agentId"  msg:"agentId"`
	ServiceType    int32  `db:"ser_type" json:"serviceType"  msg:"serviceType"`
	HostName       string `db:"host_name" json:"hostName" msg:"hostName"`
	IP4S           string `db:"ip" json:"ip" msg:"ip"`
	StartTimestamp int64  `db:"start_time" json:"startTimestamp" msg:"startTimestamp"`
	EndTimestamp   int64  `db:"end_time" json:"end_time" msg:"end_time"`
	IsContainer    bool   `db:"is_container" json:"is_container" msg:"is_container"`
	OperatingEnv   int32  `db:"operating_env" json:"operating_env" msg:"operating_env"`
	AgentInfo      string `db:"agent_info" json:"agent_info" msg:"agent_info"`
}

// NewAgentInfo ...
func NewAgentInfo() *AgentInfo {
	return &AgentInfo{}
}
