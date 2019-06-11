package channel

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bsed/trace/pkg/alert"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

// Email 邮件
type Email struct {
	client   *fasthttp.Client
	logger   *zap.Logger
	addr     string
	centerID string
	subject  string
}

// NewEmail ...
func NewEmail(l *zap.Logger, addr, centerID, subject string) *Email {
	return &Email{
		client:   &fasthttp.Client{},
		logger:   l,
		addr:     addr,
		centerID: centerID,
		subject:  subject,
	}
}

// AlertPush ...
func (e *Email) AlertPush(msg *alert.Alert) error {
	var args fasthttp.Args
	args.Set("messagecenterid", e.centerID)
	args.Set("typeOfMessageCenter", "email")

	b, _ := json.Marshal(msg)
	args.Set("model", string(b))
	args.Set("subject", e.subject)
	// args.Set("sign", "1")
	for _, addr := range msg.Addrs {
		args.Set("to", fmt.Sprintf("[%s]", addr))
		_, body, err := e.client.Post(nil, e.addr, &args)
		if err != nil {
			e.logger.Error("email http Post", zap.Any("error", err.Error()))
			continue
		}
		if !strings.Contains(string(body), "success") {
			e.logger.Error("email retrun err", zap.Error(fmt.Errorf("%s", string(body))))
			continue
		}
	}
	return nil
}
