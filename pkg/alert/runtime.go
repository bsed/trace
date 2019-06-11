package alert

// Runtimes ...
type Runtimes struct {
	Runtimes map[string]*Runtime `msg:"rs"` // agentid为key
}

// NewRuntimes ...
func NewRuntimes() *Runtimes {
	return &Runtimes{
		Runtimes: make(map[string]*Runtime),
	}
}

// Runtime ...
type Runtime struct {
	JVMCpuload    float64 `msg:"jc"` // jvm cpuload
	SystemCpuload float64 `msg:"sc"` // system cpuload
	JVMHeap       int64   `msg:"jh"` // jvm heap
	Count         int     `msg:"c"`  // 计数包的个数
}

// NewRuntime ...
func NewRuntime() *Runtime {
	return &Runtime{}
}
