package control

import "sync"

// Apps apps
type Apps struct {
	sync.RWMutex
	apps map[string]*App
}

func newApps() *Apps {
	return &Apps{
		apps: make(map[string]*App),
	}
}

func (a *Apps) get(appName string) (*App, bool) {
	a.RLock()
	app, ok := a.apps[appName]
	a.RUnlock()
	return app, ok
}

func (a *Apps) add(appName string, app *App) {
	a.Lock()
	a.apps[appName] = app
	a.Unlock()
}
