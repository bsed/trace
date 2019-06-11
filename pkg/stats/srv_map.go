package stats

// SrvMap 应用拓扑
type SrvMap struct {
	AppType      int16                        // 本服务服务类型
	UnknowParent *UnknowParent                // 未接入监控的请求者
	Targets      map[int16]map[string]*Target // 子节点拓扑图
	// Parents      map[string]*SrvParent        // 展示所有父节点
}

// NewSrvMap ...
func NewSrvMap() *SrvMap {
	return &SrvMap{
		UnknowParent: NewUnknowParent(),
		Targets:      make(map[int16]map[string]*Target),
		// Parents:      make(map[string]*SrvParent),
	}
}

// UnknowParent 未接入监控的服务，只能抓到访问地址
type UnknowParent struct {
	AccessCount    int   // 访问总数
	AccessDuration int32 // 访问总耗时
}

// NewUnknowParent ...
func NewUnknowParent() *UnknowParent {
	return &UnknowParent{}
}

// SrvParent 父节点访问子节点信息
type SrvParent struct {
	Type           int16
	TargetCount    int   // 目标应用收到请求总数
	TargetErrCount int   // 目标应用内部异常数
	AccessDuration int32 // 访问总耗时
}

// NewSrvParent ....
func NewSrvParent() *SrvParent {
	return &SrvParent{}
}

// Target ...
type Target struct {
	AccessCount    int   // 访问总数
	AccessErrCount int   // 访问错误数
	AccessDuration int32 // 访问总耗时
}

// NewTarget ...
func NewTarget() *Target {
	return &Target{}
}
