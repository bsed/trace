package control

import "sync"

// Sql ...
type Sql struct {
	sync.RWMutex
	alerts map[int]*Alert // key为告警类型，value为告警
}

func newSql() *Sql {
	return &Sql{
		alerts: make(map[int]*Alert),
	}
}

// addAlert 添加告警记录
func (s *Sql) addAlert(alertType int, alert *Alert) {
	s.Lock()
	s.alerts[alertType] = alert
	s.Unlock()
}

// getAlert 获取alert
func (s *Sql) getAlert(alertType int) (*Alert, bool) {
	s.RLock()
	alert, ok := s.alerts[alertType]
	s.RUnlock()
	return alert, ok
}

// Sqls ...
type Sqls struct {
	sync.RWMutex
	Sqls map[string]*Sql
}

func (s *Sqls) get(sqlID string) (*Sql, bool) {
	s.RLock()
	sql, ok := s.Sqls[sqlID]
	s.RUnlock()
	return sql, ok
}

func (s *Sqls) add(sqlID string, sql *Sql) {
	s.Lock()
	s.Sqls[sqlID] = sql
	s.Unlock()
}

func newSqls() *Sqls {
	return &Sqls{
		Sqls: make(map[string]*Sql),
	}
}
