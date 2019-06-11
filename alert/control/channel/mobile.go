package channel

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bsed/trace/pkg/alert"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

// Mobile 手机短信
type Mobile struct {
	logger   *zap.Logger
	client   *fasthttp.Client
	addr     string
	centerID string
}

// NewMobile ...
func NewMobile(l *zap.Logger, addr, centerID string) *Mobile {
	return &Mobile{
		client:   &fasthttp.Client{},
		logger:   l,
		addr:     addr,
		centerID: centerID,
	}
}

// AlertPush ...
func (m *Mobile) AlertPush(msg *alert.Alert) error {
	var args fasthttp.Args
	args.Set("messagecenterid", m.centerID)
	args.Set("typeOfMessageCenter", "sms")
	args.Set("sign", "1")
	b, _ := json.Marshal(msg)
	args.Set("model", string(b))
	for _, addr := range msg.Addrs {
		args.Set("mobilenumber", addr)
		_, body, err := m.client.Post(nil, m.addr, &args)
		if err != nil {
			m.logger.Error("mobile http Post", zap.Any("error", err.Error()))
			continue
		}
		if !strings.Contains(string(body), "success") {
			m.logger.Error("mobile retrun err", zap.Error(fmt.Errorf("%s", string(body))))
			continue
		}
	}
	return nil
}
