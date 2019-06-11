package alert

// Alert 告警具体详情
type Alert struct {
	Channel    string   `json:"-"`          // 通知工具 email/mobile
	Type       string   `json:"Type"`       // 告警类型 告警/告警恢复
	Detail     string   `json:"Detail"`     // 告警概述 appName/告警类型/数据+单位
	ID         string   `json:"Id"`         // 告警ID
	Addrs      []string `json:"-"`          // 手机号码或者邮箱地址
	DetailAddr string   `json:"DetailAddr"` // 详情地址 http://apmtest.tf56.lo/ui/alerts/history?id=1558578065702692000
	Time       string   `json:"Timestamp"`
}

// NewAlert ...
func NewAlert() *Alert {
	return &Alert{}
}

// <APM告警/告警恢复>
// 概述：helm/接口错误率/30百分比
// id: 1558578065702692000
// 详情地址：http://apmtest.tf56.lo/ui/alerts/history?id=1558578065702692000
