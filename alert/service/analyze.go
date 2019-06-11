package service

import (
	"github.com/bsed/trace/pkg/alert"
	"github.com/bsed/trace/pkg/constant"
	"github.com/nats-io/nats.go"
	"github.com/vmihailenco/msgpack"
	"go.uber.org/zap"
)

func msgHandle(msg *nats.Msg) {
	data := alert.NewData()
	if err := msgpack.Unmarshal(msg.Data, data); err != nil {
		logger.Warn("msgpack unmarshal", zap.String("error", err.Error()))
		return
	}
	switch data.Type {
	// 接口统计
	case constant.ALERT_TYPE_API:
		if err := gAlert.apps.apiRouter(data.AppName, data); err != nil {
			// logger.Debug("api route error", zap.String("error", err.Error()), zap.String("appName", data.AppName), zap.String("agentID", data.AgentID))
			return
		}
		break
	// 数据库统计
	case constant.ALERT_TYPE_SQL:
		if err := gAlert.apps.sqlRouter(data.AppName, data); err != nil {
			// logger.Debug("sql route error", zap.String("error", err.Error()), zap.String("appName", data.AppName), zap.String("agentID", data.AgentID))
			return
		}
		break
	// runtime数据
	case constant.ALERT_TYPE_RUNTIME:
		if err := gAlert.apps.runtimeRoute(data.AppName, data); err != nil {
			// logger.Debug("runtime route error", zap.String("error", err.Error()), zap.String("appName", data.AppName), zap.String("agentID", data.AgentID))
			return
		}
		break
	// 异常统计
	case constant.ALERT_TYPE_EXCEPTION:
		if err := gAlert.apps.exRouter(data.AppName, data); err != nil {
			// logger.Debug("ex route error", zap.String("error", err.Error()), zap.String("appName", data.AppName), zap.String("agentID", data.AgentID))
			return
		}
		break
	}
}

// compare 对比阀值
func compare(compareValue, alertValue float64, compare int) bool {
	switch compare {
	// 大于
	case 1:
		if compareValue > alertValue {
			return true
		}
		break
	// 小于
	case 2:
		if compareValue < alertValue {
			return true
		}
		break
	// 等于
	case 3:
		if compareValue == alertValue {
			return true
		}
		break
	}

	return false
}
