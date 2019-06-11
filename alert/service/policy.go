package service

// Policy 策略
type Policy struct {
	AppName    string   // app名
	Owner      string   // owner
	ID         string   // policyid
	Group      string   // 组
	Channel    string   // 告警方式
	Users      []string // 用户
	UpdateDate int64    // 更新时间
	checkTime  int64    // 上次检查时间
}

// newPolicy
func newPolicy() *Policy {
	return &Policy{}
}

// newPolicy
func newAlertInfo() *AlertInfo {
	return &AlertInfo{}
}

// AlertInfo 策略信息
type AlertInfo struct {
	Type     int      // 监控项类型
	Compare  int      // 比较类型 1: > 2:<  3:=
	Duration int      // 持续时间, 1 代表1分钟
	Keys     []string // code...
	Value    float64  // 阀值
	Unit     string   //单位
}

// SpecialAlert 特殊监控类型
type SpecialAlert struct {
	API map[string]map[int]*AlertInfo
}

func newSpecialAlert() *SpecialAlert {
	return &SpecialAlert{
		API: make(map[string]map[int]*AlertInfo),
	}
}
