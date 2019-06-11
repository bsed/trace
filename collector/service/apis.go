package service

import "sync"

// DubboAPIMap ...
type DubboAPIMap struct {
	sync.RWMutex
	APIs map[string]string
}

// NewDubboAPIMap ...
func NewDubboAPIMap() *DubboAPIMap {
	return &DubboAPIMap{
		APIs: make(map[string]string),
	}
}

// Get ...
func (d *DubboAPIMap) Get(api string) (string, bool) {
	d.RLock()
	appName, ok := d.APIs[api]
	d.RUnlock()
	return appName, ok
}

// Add ...
func (d *DubboAPIMap) Add(api, appName string) {
	d.Lock()
	d.APIs[api] = appName
	d.Unlock()
}
