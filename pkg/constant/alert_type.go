package constant

var Alert map[string]int
var AlertInfo map[int]string

const (
	ALERT_APM_API_ERROR_RATIO = 1 // 接口访问错误率
	ALERT_APM_API_ERROR_COUNT = 2 // 接口访问错误次数
	ALERT_APM_EXCEPTION_RATIO = 3 // 内部异常率
	ALERT_APM_SQL_ERROR_RATIO = 4 // sql错误率
	ALERT_APM_API_DURATION    = 5 // 接口平均耗时
	ALERT_APM_API_COUNT       = 6 // 接口访问次数
	ALERT_APM_CPU_USED_RATIO  = 7 // cpu使用率
	ALERT_APM_MEM_USED_RATION = 8 // JVM Heap使用量

	ALERT_TYPE_API       = 1000 // api 数据
	ALERT_TYPE_SQL       = 1001 // sql 数据
	ALERT_TYPE_RUNTIME   = 1002 // runtime 数据
	ALERT_TYPE_EXCEPTION = 1003 // 异常 数据

	POLICY_Type_DEFAULT = 1 // 默认模版
	POLICY_Type_CUSTOM  = 2 // 自定义策略模版
)

func initAlertType() {
	Alert = make(map[string]int)
	AlertInfo = make(map[int]string)

	Alert["apm.api_error.ratio"] = 1
	AlertInfo[1] = "接口访问错误率"

	Alert["apm.api_error.count"] = 2
	AlertInfo[2] = "接口访问错误次数"

	Alert["apm.exception.ratio"] = 3
	AlertInfo[3] = "内部异常率"

	Alert["apm.sql_error.ratio"] = 4
	AlertInfo[4] = "sql错误率"

	Alert["apm.api.duration"] = 5
	AlertInfo[5] = "接口平均耗时"

	Alert["apm.api.count"] = 6
	AlertInfo[6] = "接口访问次数"

	Alert["system.cpu_used.ratio"] = 7
	AlertInfo[7] = "cpu使用率"

	Alert["system.mem_used.ratio"] = 8
	AlertInfo[8] = "JVM Heap使用量"
}

// AlertType 通过描述获取类型
func AlertType(desc string) (int, bool) {
	alertType, ok := Alert[desc]
	return alertType, ok
}

// AlertDesc 通过类型获取描述
func AlertDesc(alertType int) (string, bool) {
	desc, ok := AlertInfo[alertType]
	return desc, ok
}
