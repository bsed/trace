package service

import (
	"github.com/bsed/trace/pkg/alert"
	"github.com/bsed/trace/pkg/constant"
	"github.com/nats-io/nats.go"
	"github.com/vmihailenco/msgpack"
	"go.uber.org/zap"
)

func msgHandle(msg *nats.Msg) {
	packet := alert.NewData()
	if err := msgpack.Unmarshal(msg.Data, packet); err != nil {
		logger.Warn("msgpack unmarshal", zap.String("error", err.Error()))
		return
	}
	switch packet.Type {
	case constant.ALERT_TYPE_API:
		if err := gCollector.apps.routerApi(packet); err != nil {
			logger.Warn("routerApi error", zap.String("error", err.Error()))
			break
		}
		break
	default:
		logger.Warn("msgHandle unknow type", zap.Int("Type", packet.Type))
		break
	}
}
