package control

import (
	"sync"
)

// Api api告警信息
type Api struct {
	sync.RWMutex
	alerts map[int]*Alert // key为告警类型，value为告警
}

func newApi() *Api {
	return &Api{
		alerts: make(map[int]*Alert),
	}
}

// addAlert 添加告警记录
func (a *Api) addAlert(alertType int, alert *Alert) {
	a.Lock()
	a.alerts[alertType] = alert
	a.Unlock()
}

// getAlert 获取alert
func (a *Api) getAlert(alertType int) (*Alert, bool) {
	a.RLock()
	alert, ok := a.alerts[alertType]
	a.RUnlock()
	return alert, ok
}

// Apis ...
type Apis struct {
	sync.RWMutex
	Apis map[string]*Api
}

func (a *Apis) get(apiStr string) (*Api, bool) {
	a.RLock()
	api, ok := a.Apis[apiStr]
	a.RUnlock()
	return api, ok
}

func (a *Apis) add(apiStr string, api *Api) {
	a.Lock()
	a.Apis[apiStr] = api
	a.Unlock()
}

func newApis() *Apis {
	return &Apis{
		Apis: make(map[string]*Api),
	}
}
