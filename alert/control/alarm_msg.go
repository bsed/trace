package control

// AlarmMsg 告警信息
type AlarmMsg struct {
	AppName        string   // 应用名
	AgentID        string   // 应用ID
	Type           int      // 告警类型
	API            string   // api
	SQL            string   // sql
	ThresholdValue float64  // 阀值
	AlertValue     float64  // 告警值
	Channel        string   // 告警通道/告警工具
	Users          []string // 告警对象
	Time           int64    // 告警时间
	Interval       int64    // 告警间隔,单位秒
	IsRecovery     bool     //是否需要恢复告警
	Unit           string   // 单位
	ID             int64    // 告警id
}
