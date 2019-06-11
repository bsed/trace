package stats

import "github.com/bsed/trace/pkg/pinpoint/thrift/pinpoint"

// JVMStats jvm 信息计算统计
type JVMStats struct {
	Agents map[string]*JVMInfo
}

// NewJVMStats ...
func NewJVMStats() *JVMStats {
	return &JVMStats{
		Agents: make(map[string]*JVMInfo),
	}
}

// JVMInfo ...
type JVMInfo struct {
	CPULoad *JVMCPULoad `json:"cpuload"`
	GC      *JVMGC      `json:"gc"`
}

// NewJVMInfo ...
func NewJVMInfo() *JVMInfo {
	return &JVMInfo{
		CPULoad: &JVMCPULoad{},
		GC:      &JVMGC{},
	}
}

// JVMCPULoad ...
type JVMCPULoad struct {
	Jvm    float64 `json:"jvm"`
	System float64 `json:"system"`
}

// JVMGC ...
type JVMGC struct {
	Type                     pinpoint.TJvmGcType `json:"type"`
	HeapUsed                 int64               `json:"heapUsed"`
	HeapMax                  int64               `json:"heapMax"`
	NonHeapUsed              int64               `json:"-"`
	NonHeapMax               int64               `json:"-"`
	GcOldCount               int64               `json:"jvmGcOldCount"`
	JvmGcOldTime             int64               `json:"jvmGcOldTime"`
	JvmGcNewCount            int64               `json:"jvmGcNewCount"`
	JvmGcNewTime             int64               `json:"jvmGcNewTime"`
	JvmPoolCodeCacheUsed     float64             `json:"-"`
	JvmPoolNewGenUsed        float64             `json:"-"`
	JvmPoolOldGenUsed        float64             `json:"-"`
	JvmPoolSurvivorSpaceUsed float64             `json:"-"`
	JvmPoolPermGenUsed       float64             `json:"JvmPoolPermGenUsed"`
	JvmPoolMetaspaceUsed     float64             `json:"JvmPoolMetaspaceUsed"`
}
