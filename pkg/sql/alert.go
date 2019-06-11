package sql

// 加载所有策略
var LoadPolicys string = `SELECT name, owner, api_alerts, channel, group,
 policy_id, update_date, users FROM alerts_app ;`

// 加载策略详情
var LoadAlert string = `SELECT alerts FROM alerts_policy WHERE id=?;`

var InsertAPIAlarmHistory string = `INSERT INTO alarm_history (app_name, agent_id, host_name, alert_name, alert_value, alert_unit,
	alert_compare, alarm_value, input_date, api_sql) VALUES (?,?,?,?,?,?,?,?,?,?);`

var LoadUers string = `SELECT id, email, mobile FROM account ;`

var InsertApiAlert string = `INSERT INTO alert_history (const_id, id, app_name,
	type, api,  alert, alert_value, channel, users, input_date) VALUES (?,?,?,?,?,?,?,?,?,?);`

var InsertSQLAlert string = `INSERT INTO alert_history (const_id, id, app_name, 
		type, sql,  alert, alert_value, channel, users, input_date) VALUES (?,?,?,?,?,?,?,?,?,?);`

var InsertRuntimeAlert string = `INSERT INTO alert_history (const_id, id, app_name, 
			type, agent_id,  alert, alert_value, channel, users, input_date) VALUES (?,?,?,?,?,?,?,?,?,?);`

// 加载默认策略详情
var LoaddefaultPolicy string = `SELECT alerts FROM alerts_policy WHERE name='apm-default-policy' ALLOW FILTERING;`
