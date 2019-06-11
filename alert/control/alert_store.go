package control

import (
	"fmt"
	"time"

	"github.com/bsed/trace/pkg/constant"
	"github.com/bsed/trace/pkg/sql"
	"github.com/bsed/trace/pkg/util"
	"go.uber.org/zap"
)

// apiAlarmStore ....
func (c *Control) apiAlarmStore(msg *AlarmMsg, alertIndex int, alertType bool) error {
	cql := c.getCql()
	if cql == nil {
		logger.Warn("get cql failed")
		return fmt.Errorf("get cql failed")
	}
	alertName, _ := constant.AlertDesc(alertIndex)
	tmpAlert := &util.Alert{
		Name: alertName,
	}
	var alertTypeInt int
	if alertType {
		alertTypeInt = 2
	} else {
		alertTypeInt = 1
	}

	query := cql.Query(sql.InsertApiAlert,
		1,
		msg.ID,
		msg.AppName,
		alertTypeInt,
		msg.API,
		tmpAlert,
		msg.AlertValue,
		msg.Channel,
		msg.Users,
		time.Now().Unix(),
	)

	if err := query.Exec(); err != nil {
		logger.Warn("alarm store", zap.String("SQL", query.String()), zap.String("error", err.Error()))
		return err
	}
	return nil
}

func (c *Control) sqlAlarmStore(msg *AlarmMsg, alertIndex int, alertType bool) error {
	cql := c.getCql()
	if cql == nil {
		logger.Warn("get cql failed")
		return fmt.Errorf("get cql failed")
	}
	alertName, _ := constant.AlertDesc(alertIndex)
	tmpAlert := &util.Alert{
		Name: alertName,
	}
	var alertTypeInt int
	if alertType {
		alertTypeInt = 2
	} else {
		alertTypeInt = 1
	}
	query := cql.Query(sql.InsertSQLAlert,
		1,
		msg.ID,
		msg.AppName,
		alertTypeInt,
		msg.SQL,
		tmpAlert,
		msg.AlertValue,
		msg.Channel,
		msg.Users,
		time.Now().Unix(),
	)

	if err := query.Exec(); err != nil {
		logger.Warn("alarm store", zap.String("SQL", query.String()), zap.String("error", err.Error()))
		return err
	}
	return nil
}

func (c *Control) runtimeAlarmStore(msg *AlarmMsg, alertIndex int, alertType bool) error {
	cql := c.getCql()
	if cql == nil {
		logger.Warn("get cql failed")
		return fmt.Errorf("get cql failed")
	}
	alertName, _ := constant.AlertDesc(alertIndex)
	tmpAlert := &util.Alert{
		Name: alertName,
	}
	var alertTypeInt int
	if alertType {
		alertTypeInt = 2
	} else {
		alertTypeInt = 1
	}

	query := cql.Query(sql.InsertRuntimeAlert,
		1,
		msg.ID,
		msg.AppName,
		alertTypeInt,
		msg.AgentID,
		tmpAlert,
		msg.AlertValue,
		msg.Channel,
		msg.Users,
		time.Now().Unix(),
	)

	if err := query.Exec(); err != nil {
		logger.Warn("alarm store", zap.String("SQL", query.String()), zap.String("error", err.Error()))
		return err
	}
	return nil
}

func (c *Control) exAlarmStore(msg *AlarmMsg, alertIndex int, alertType bool) error {
	var InsertAPIAlertHistory string = `INSERT INTO alert_history (const_id, id, app_name, 
		type, api,  alert, alert_value, channel, users, input_date) VALUES (?,?,?,?,?,?,?,?,?,?);`
	cql := c.getCql()
	if cql == nil {
		logger.Warn("get cql failed")
		return fmt.Errorf("get cql failed")
	}
	alertName, _ := constant.AlertDesc(alertIndex)
	tmpAlert := &util.Alert{
		Name: alertName,
	}
	var alertTypeInt int
	if alertType {
		alertTypeInt = 2
	} else {
		alertTypeInt = 1
	}

	query := cql.Query(InsertAPIAlertHistory,
		1,
		msg.ID,
		msg.AppName,
		alertTypeInt,
		"",
		tmpAlert,
		msg.AlertValue,
		msg.Channel,
		msg.Users,
		time.Now().Unix(),
	)

	if err := query.Exec(); err != nil {
		logger.Warn("alarm store", zap.String("SQL", query.String()), zap.String("error", err.Error()))
		return err
	}
	return nil
}
