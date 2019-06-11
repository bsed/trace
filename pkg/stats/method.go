package stats

// Methods 接口统计
type Methods struct {
	ApiStr  string
	Methods map[int32]*Method
}

// NewMethods ...
func NewMethods() *Methods {
	return &Methods{
		Methods: make(map[int32]*Method),
	}
}

// Get 获取medthod信息
func (m *Methods) Get(methodID int32) (*Method, bool) {
	method, ok := m.Methods[methodID]
	return method, ok
}

// Store 存储method信息
func (m *Methods) Store(methodID int32, method *Method) {
	m.Methods[methodID] = method
}

// Method 接口信息
type Method struct {
	Type        int16 // 服务类型
	Duration    int32 // 总耗时
	Count       int   // 发生次数
	ErrCount    int   // 错误次数
	MinDuration int32 // 最小耗时
	MaxDuration int32 // 最大耗时
}

// NewMethod ...
func NewMethod(methodType int16) *Method {
	return &Method{
		Type: methodType,
	}
}
