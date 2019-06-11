package alert

// Exception 异常计数
type Exception struct {
	Count    int `msg:"c"`
	ErrCount int `msg:"ec"`
}

// NewException ...
func NewException() *Exception {
	return &Exception{}
}
