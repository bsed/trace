package stats

// ApiStore api集合
type ApiStore struct {
	// apps
	Apps map[string]*App
}

func NewApiStore() *ApiStore {
	return &ApiStore{
		Apps: make(map[string]*App),
	}
}

// App apis
type App struct {
	Urls   map[string]*Url   `msg:"url"`   // url 统计信息
	Dubbos map[string]*Dubbo `msg:"dubbo"` // dubbo统计信息
}

func NewApp() *App {
	return &App{
		Urls:   make(map[string]*Url),
		Dubbos: make(map[string]*Dubbo),
	}
}

// Dubbo ...
type Dubbo struct {
	Duration          int32 `msg:"d"`   // 总耗时
	MinDuration       int32 `msg:"min"` // 最小耗时
	MaxDuration       int32 `msg:"max"` // 最大耗时
	AccessCount       int   `msg:"ac"`  // 访问总数
	AccessErrCount    int   `msg:"aec"` // 访问错误数
	SatisfactionCount int   `msg:"sc"`  // 满意次数
	TolerateCount     int   `msg:"tc"`  // 可容忍次数
}

// NewDubbo new dubbo
func NewDubbo() *Dubbo {
	return &Dubbo{}
}

// Url url
type Url struct {
	Duration          int32 `msg:"d"`   // 总耗时
	MinDuration       int32 `msg:"min"` // 最小耗时
	MaxDuration       int32 `msg:"max"` // 最大耗时
	AccessCount       int   `msg:"ac"`  // 访问总数
	AccessErrCount    int   `msg:"aec"` // 访问错误数
	SatisfactionCount int   `msg:"sc"`  // 满意次数
	TolerateCount     int   `msg:"tc"`  // 可容忍次数
}

// NewUrl new url
func NewUrl() *Url {
	return &Url{}
}
