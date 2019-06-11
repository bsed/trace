package misc

import (
	"strings"
	"time"

	"github.com/imdevlab/g"

	"github.com/gocql/gocql"
	"github.com/imdevlab/g/utils"
	"github.com/labstack/echo"
)

var StaticCql *gocql.Session
var TraceCql *gocql.Session

// 获取开始和截止日期
func StartEndDate(c echo.Context) (start time.Time, end time.Time, err error) {
	startRaw := c.FormValue("start")
	endRaw := c.FormValue("end")

	// start和end的时间字符串转成秒级时间戳:2019-01-10 00:00:00
	start, err = utils.StringToTime(startRaw)
	if err != nil {
		return
	}

	end, err = utils.StringToTime(endRaw)
	if err != nil {
		return
	}

	// utils.OnlyAlphaAndNum
	return
}

// 切分pinpoint采集的java method，获得class和method name
// e.g. org.apache.catalina.core.StandardHostValve.invoke(org.apache.catalina.connector.Request request, org.apache.catalina.connector.Response response)
func SplitMethod(m string) (string, string) {
	n := strings.Index(m, "(")
	for i := n; i >= 0; i-- {
		if m[i] == '.' {
			return m[:i], m[i+1:]
		}
	}
	return "", m
}

func Timestamp2TimeString(t int64) string {
	tm, _ := utils.MSToTime(t)
	return tm.Format("2006-01-02 15:04:05.999")
}

func GetMethodByID(appName string, id int) string {
	q := StaticCql.Query(`SELECT method_info FROM app_methods WHERE app_name = ? and method_id=?`, appName, id)
	var method string
	err := q.Scan(&method)
	if err != nil {
		return "method_not_found"
	}

	return method
}

func GetSqlByID(appName string, id int) string {
	q := StaticCql.Query(`SELECT sql_info FROM app_sqls WHERE app_name=? AND  sql_id=?`, appName, id)
	var sql string
	err := q.Scan(&sql)
	if err != nil {
		return "sql_not_found"
	}

	b, _ := g.B64.DecodeString(sql)
	b = utils.TrimBytesExtraLineAndSpace(b)
	return utils.Bytes2String(b)
}

func GetExceptionByID(appName string, id int) string {
	q := StaticCql.Query(`SELECT str_info FROM app_strs WHERE app_name=? AND  str_id=?`, appName, id)
	var class string
	err := q.Scan(&class)
	if err != nil {
		return "ex_not_found"
	}

	return class
}

func TimeToChartString(t time.Time) string {
	return t.Format("01-02 15:04")
}
func TimeToChartString1(t time.Time) string {
	return t.Format("01-02 15:04:05")
}
func TimeToChartString2(t time.Time) string {
	return t.Format("06-01-02 15:04")
}
