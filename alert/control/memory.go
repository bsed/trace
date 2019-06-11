package control

import "sync"

// Memory ...
type Memory struct {
	sync.RWMutex
	alerts map[int]*Alert
}

func newMemory() *Memory {
	return &Memory{
		alerts: make(map[int]*Alert),
	}
}

// addAlert 添加告警记录
func (m *Memory) addAlert(alertType int, alert *Alert) {
	m.Lock()
	m.alerts[alertType] = alert
	m.Unlock()
}

// getAlert 获取alert
func (m *Memory) getAlert(alertType int) (*Alert, bool) {
	m.RLock()
	alert, ok := m.alerts[alertType]
	m.RUnlock()
	return alert, ok
}

// Memorys ..
type Memorys struct {
	sync.RWMutex
	Agents map[string]*Memory
}

func newMemorys() *Memorys {
	return &Memorys{
		Agents: make(map[string]*Memory),
	}
}

func (m *Memorys) get(agentID string) (*Memory, bool) {
	m.RLock()
	agent, ok := m.Agents[agentID]
	m.RUnlock()
	return agent, ok
}

func (m *Memorys) add(agentID string, cpu *Memory) {
	m.Lock()
	m.Agents[agentID] = cpu
	m.Unlock()
}
