package control

import (
	"fmt"
	"strings"

	"github.com/bsed/trace/alert/control/channel"
	"github.com/bsed/trace/pkg/alert"
	"go.uber.org/zap"
)

// Channels 通知工具集合
type Channels struct {
	Channels map[string]Channel
}

// newChannels ...
func newChannels() *Channels {
	return &Channels{
		Channels: make(map[string]Channel),
	}
}

// Channel 通知工具
type Channel interface {
	AlertPush(msg *alert.Alert) error
}

// addChannel 添加告警通道
func (c *Channels) addChannel(channelName string) error {
	if strings.EqualFold(channelName, "email") {
		email := channel.NewEmail(logger, gControl.conf.EmailURL, gControl.conf.EmaiCentID, gControl.conf.EmailSubject)
		c.Channels["email"] = email
	} else if strings.EqualFold(channelName, "mobile") {
		mobile := channel.NewMobile(logger, gControl.conf.Mobileurl, gControl.conf.MobileCentID)
		c.Channels["mobile"] = mobile
	}
	return nil
}

func (c *Channels) alertPush(channelName string, alert *alert.Alert) error {
	channel, ok := c.Channels[channelName]
	if !ok {
		logger.Warn("unfind channel", zap.String("channel name", channelName))
		return fmt.Errorf("unfind channel, channel name is %s", channelName)
	}
	err := channel.AlertPush(alert)
	if err != nil {
		logger.Warn("alert push failed", zap.String("channel name", channelName), zap.Error(err))
		return err
	}
	return nil
}
