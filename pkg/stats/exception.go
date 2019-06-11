package stats

// Exceptions 异常统计
type Exceptions struct {
	Count     int // span发送的次数
	ErrCount  int // 异常总数
	ExMethods map[int32]*ExMethod
}

// NewExceptions ...
func NewExceptions() *Exceptions {
	return &Exceptions{
		ExMethods: make(map[int32]*ExMethod),
	}
}

// Get 获取Method异常信息
func (a *Exceptions) Get(methodID int32) (*ExMethod, bool) {
	exMethod, ok := a.ExMethods[methodID]
	return exMethod, ok
}

// Store 存储methodID异常信息
func (a *Exceptions) Store(methodID int32, exMethod *ExMethod) {
	a.ExMethods[methodID] = exMethod
}

// ExMethod 接口信息
type ExMethod struct {
	Exceptions map[int32]*Exception
}

// Get ...
func (e *ExMethod) Get(exID int32) (*Exception, bool) {
	exception, ok := e.Exceptions[exID]
	return exception, ok
}

// Store ...
func (e *ExMethod) Store(exID int32, ex *Exception) {
	e.Exceptions[exID] = ex
}

// NewExMethod 新接口
func NewExMethod() *ExMethod {
	return &ExMethod{
		Exceptions: make(map[int32]*Exception),
	}
}

// NewException ...
func NewException() *Exception {
	return &Exception{}
}

// Exception 异常接口
type Exception struct {
	Type        int   // 服务类型
	Duration    int32 // 总耗时
	Count       int   // 发生次数
	MinDuration int32 // 最小耗时
	MaxDuration int32 // 最大耗时
}
