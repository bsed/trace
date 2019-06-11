package stats

// Runtimes runtimes计算
type Runtimes struct {
	Runtimes map[string]*Runtime // agentid为key
}

// NewRuntimes ...
func NewRuntimes() *Runtimes {
	return &Runtimes{
		Runtimes: make(map[string]*Runtime),
	}
}

// Runtime ...
type Runtime struct {
	JVMCpuload    float64 // jvm cpuload
	SystemCpuload float64 // system cpuload
	JVMHeap       int64   // jvm heap
	Count         int     // 记录包数
}

// NewRuntime ...
func NewRuntime() *Runtime {
	return &Runtime{}
}
