package service

// System 系统信息采集服务
type System struct {
}

func newSystem() *System {
	return &System{}
}

// Start 启动系统采集服务
func (s *System) Start() error {
	return nil
}

// Close 关闭系统采集服务
func (s *System) Close() error {
	return nil
}
