package control

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/imdevlab/g/utils"

	"github.com/gocql/gocql"
	"github.com/bsed/trace/pkg/alert"
	"github.com/bsed/trace/pkg/constant"
	"go.uber.org/zap"
)

var logger *zap.Logger

// Control 告警控制中心
type Control struct {
	conf     *Conf     // 配置文件
	Apps     *Apps     // 应用缓存
	channels *Channels // 通知工具
	users    *Users
	getCql   func() *gocql.Session
}

var gControl *Control

// New new control
func New(conf *Conf, zlog *zap.Logger) *Control {
	logger = zlog
	control := &Control{
		conf:     conf,
		Apps:     newApps(),
		channels: newChannels(),
		users:    newUsers(),
	}
	gControl = control
	return control
}

// Init init
func (c *Control) Init(f func() *gocql.Session) error {
	for _, channelName := range c.conf.Channels {
		c.channels.addChannel(channelName)
	}
	c.getCql = f
	return nil
}

// AlertPush 告警推送
func (c *Control) AlertPush(msg *AlarmMsg) error {
	isPush := false
	isRecovery := false
	app, ok := c.Apps.get(msg.AppName)
	if !ok {
		app = newApp()
		c.Apps.add(msg.AppName, app)
	}
	switch msg.Type {
	// 接口访问错误率
	case constant.ALERT_APM_API_ERROR_RATIO:
		isPush, isRecovery = app.checkApi(msg)
		if isPush {
			c.apiAlarmStore(msg, constant.ALERT_APM_API_ERROR_RATIO, isRecovery)
		}
		break
	// 接口访问错误次数
	case constant.ALERT_APM_API_ERROR_COUNT:
		isPush, isRecovery = app.checkApi(msg)
		if isPush {
			c.apiAlarmStore(msg, constant.ALERT_APM_API_ERROR_COUNT, isRecovery)
		}
		break
	// 内部异常率
	case constant.ALERT_APM_EXCEPTION_RATIO:
		isPush, isRecovery = app.checkEx(msg)
		if isPush {
			c.exAlarmStore(msg, constant.ALERT_APM_EXCEPTION_RATIO, isRecovery)
		}
		break
	// sql错误率
	case constant.ALERT_APM_SQL_ERROR_RATIO:
		isPush, isRecovery = app.checkSql(msg)
		if isPush {
			c.sqlAlarmStore(msg, constant.ALERT_APM_SQL_ERROR_RATIO, isRecovery)
		}
		break
	// 接口平均耗时
	case constant.ALERT_APM_API_DURATION:
		isPush, isRecovery = app.checkApi(msg)
		if isPush {
			c.apiAlarmStore(msg, constant.ALERT_APM_API_DURATION, isRecovery)
		}
		break
	// 接口访问次数
	case constant.ALERT_APM_API_COUNT:
		isPush, isRecovery = app.checkApi(msg)
		if isPush {
			c.apiAlarmStore(msg, constant.ALERT_APM_API_COUNT, isRecovery)
		}
		break
	// cpu使用率
	case constant.ALERT_APM_CPU_USED_RATIO:
		isPush, isRecovery = app.checkCpu(msg)
		if isPush {
			c.runtimeAlarmStore(msg, constant.ALERT_APM_CPU_USED_RATIO, isRecovery)
		}
		break
	// JVM Heap使用量
	case constant.ALERT_APM_MEM_USED_RATION:
		isPush, isRecovery = app.checkMemory(msg)
		if isPush {
			c.runtimeAlarmStore(msg, constant.ALERT_APM_MEM_USED_RATION, isRecovery)
		}
		break
	}

	if isPush {
		alert := alert.NewAlert()
		alert.Channel = msg.Channel
		// 告警类型 告警/告警恢复
		if isRecovery {
			alert.Type = "告警恢复"
		} else {
			alert.Type = "告警"
		}
		// 告警概述
		alertTypeDesc, _ := constant.AlertDesc(msg.Type)
		alert.Detail = msg.AppName + "/" + alertTypeDesc + "/" + fmt.Sprintf("%0.2f", msg.AlertValue) + "/" + msg.Unit // 告警概述 appName/告警类型/数据+单位
		alert.ID = fmt.Sprintf("%d", msg.ID)                                                                           // 告警ID
		// alert.Users = append(alert.Users, msg.Users...)                                                             // 手机号码或者邮箱地址
		for _, userID := range msg.Users {
			user, ok := gControl.users.get(userID)
			if ok {
				if msg.Channel == "email" {
					alert.Addrs = append(alert.Addrs, user.Email)
				} else {
					alert.Addrs = append(alert.Addrs, user.Mobile)
				}
			}
		}
		alert.DetailAddr = fmt.Sprintf("%s%d", gControl.conf.DetailAddr, msg.ID) // 详情地址 http://apmtest.tf56.lo/ui/alerts/history?id=1558578065702692000
		alert.Time = utils.Time2StringSecond(time.Now())

		b, _ := json.Marshal(alert)
		logger.Info("告警信息", zap.String("msg", string(b)))
		// if err := gControl.channels.alertPush(msg.Channel, alert); err != nil {
		// 	logger.Warn("push alert", zap.Error(err))
		// 	return err
		// }
	}
	return nil
}

// AddUser ...
func (c *Control) AddUser(id, email, mobile string) {
	c.users.add(id, email, mobile)
}
