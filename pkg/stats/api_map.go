package stats

// ApiMap api被调用情况
type ApiMap struct {
	Apis map[string]*Api
}

// NewApiMap ...
func NewApiMap() *ApiMap {
	return &ApiMap{
		Apis: make(map[string]*Api),
	}
}

// Api 调用信息
type Api struct {
	Parents map[string]*Parent
}

// NewApi ...
func NewApi() *Api {
	return &Api{
		Parents: make(map[string]*Parent),
	}
}
