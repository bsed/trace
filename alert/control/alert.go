package control

import "time"

// Alert 告警时间记录
type Alert struct {
	isRecovery   bool  // 是否恢复
	AlertTime    int64 // 告警时间
	RecoveryTime int64 // 恢复时间
	Count        int   // 告警次数
}

// alarm 告警更新告警时间以及告警次数
func (a *Alert) alarm(alarmTime int64) {
	a.AlertTime = alarmTime
	a.isRecovery = false
	a.Count++
}

// isAlarm 是否能报警
func (a *Alert) isAlarm(alarmTime int64) bool {
	if time.Now().Unix() > (int64(a.Count)*gControl.conf.AlarmInterval + (a.AlertTime + gControl.conf.AlarmInterval)) {
		if a.Count < gControl.conf.MaxAlarmCount {
			return true
		}
	}
	return false
}

// recovery 恢复告警更新告警时间以及告警次数
func (a *Alert) recovery(alarmTime int64) {
	a.AlertTime = 0
	a.Count = 0
	a.isRecovery = true
}

func newAlert() *Alert {
	return &Alert{}
}
