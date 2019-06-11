package stats

// Parent 所有调用者信息
type Parent struct {
	Type           int16 // 父节点类型
	AccessCount    int   // 访问次数
	AccessErrCount int   // 访问失败次数
	AccessDuration int32 // 访问耗时
	ExceptionCount int   // 异常次数
}

// NewParent ...
func NewParent() *Parent {
	return &Parent{}
}
