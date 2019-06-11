package control

// Conf config
type Conf struct {
	Channels      []string
	MaxAlarmCount int    // 最大告警次数
	AlarmInterval int64  // 两次告警时间间隔
	DetailAddr    string // apm 查询详情地址
	EmailURL      string // email服务url
	EmaiCentID    string // email centID
	EmailSubject  string // 邮件主题
	Mobileurl     string // mobile服务url
	MobileCentID  string // mobile centID
}
