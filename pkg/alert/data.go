package alert

// Data 告警存储数据
type Data struct {
	Type    int    `msg:"type"`
	AppName string `msg:"name"`
	AgentID string `msg:"id"`
	Time    int64  `msg:"time"`
	Payload []byte `msg:"payload"`
}

// NewData ...
func NewData() *Data {
	return &Data{}
}
