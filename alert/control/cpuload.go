package control

import "sync"

// Cpus ..
type Cpus struct {
	sync.RWMutex
	Agents map[string]*Cpu
}

func newCpus() *Cpus {
	return &Cpus{
		Agents: make(map[string]*Cpu),
	}
}

func (c *Cpus) get(agentID string) (*Cpu, bool) {
	c.RLock()
	agent, ok := c.Agents[agentID]
	c.RUnlock()
	return agent, ok
}

func (c *Cpus) add(agentID string, cpu *Cpu) {
	c.Lock()
	c.Agents[agentID] = cpu
	c.Unlock()
}

// Cpu ...
type Cpu struct {
	sync.RWMutex
	alerts map[int]*Alert
}

func newCpu() *Cpu {
	return &Cpu{
		alerts: make(map[int]*Alert),
	}
}

// addAlert 添加告警记录
func (c *Cpu) addAlert(alertType int, alert *Alert) {
	c.Lock()
	c.alerts[alertType] = alert
	c.Unlock()
}

// getAlert 获取alert
func (c *Cpu) getAlert(alertType int) (*Alert, bool) {
	c.RLock()
	alert, ok := c.alerts[alertType]
	c.RUnlock()
	return alert, ok
}
