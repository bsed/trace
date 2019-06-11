package alert

// NewAPIs ...
func NewAPIs() *APIs {
	return &APIs{
		APIS: make(map[string]*API),
	}
}

// APIs ..
type APIs struct {
	APIS map[string]*API `msg:"apis"`
}

// API API信息
type API struct {
	Desc           string `msg:"desc"`
	Duration       int32  `msg:"d"`   // 总耗时
	AccessCount    int    `msg:"ac"`  // 访问总数
	AccessErrCount int    `msg:"aec"` // 访问错误数
}
