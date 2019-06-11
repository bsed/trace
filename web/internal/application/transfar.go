package app

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo"
)

const PlatformName = "openapm"
const PlatformUrl = "http://apmtest.tf56.lo/ui"

type QPSResult struct {
	Code    int      `json:"code"`
	Count   int      `json:"count"`
	Message string   `json:"msg"`
	Data    *QPSData `json:"data"`
}
type QPSData struct {
	PlatformName   string         `json:"platformName"`
	PlatformUrl    string         `json:"platformUrl"`
	WarningLevel   string         `json:"warningLevel"`
	WarningMessage string         `json:"warningMessage"`
	Items          map[string]int `json:"items"`
}

/* 为传化内部其它服务提供查询接口 */
func QueryPlatformStatus(c echo.Context) error {
	stats := appList(1, 1, []string{})
	total := len(stats)

	// 计算不健康应用数
	unhealth := 0
	for _, stat := range stats {
		// 错误率超过10%或者没有节点存活，认为是不健康
		if stat.ErrorPercent >= 10 || stat.Alive <= 0 {
			unhealth++
		}
	}
	level := "绿色"
	if unhealth > 5 {
		level = "红色"
	} else if unhealth > 3 {
		level = "橙色"
	} else if unhealth > 1 {
		level = "黄色"
	}

	res := &QPSResult{
		Message: "success",
		Data: &QPSData{
			PlatformName:   PlatformName,
			PlatformUrl:    PlatformUrl,
			WarningLevel:   level,
			WarningMessage: fmt.Sprintf("不健康应用超过了%d个", unhealth),
			Items: map[string]int{
				"总应用数量":   total,
				"不健康应用数量": unhealth,
			},
		},
	}

	return c.JSON(http.StatusOK, res)
}

type QASResult struct {
	Code    int      `json:"code"`
	Count   int      `json:"count"`
	Message string   `json:"msg"`
	Data    *QASData `json:"data"`
}
type QASData struct {
	PlatformName         string              `json:"platformName"`
	PlatformUrl          string              `json:"platformUrl"`
	ItemNameList         []string            `json:"itemNameList"`
	MonitorAppStatusList []*MonitorAppStatus `json:"monitorAppStatusList"`
}

type MonitorAppStatus struct {
	AppName      string                  `json:"applicationName"`
	WarningLevel string                  `json:"appWarningLevel"`
	Items        []*MonitorAppStatusItem `json:"items"`
}

type MonitorAppStatusItem struct {
	Name  string `json:"itemName"`
	Value string `json:"itemValue"`
	// Level string `json:"itemWarningLevel"`
}

func QueryAppBaseStatusList(c echo.Context) error {
	stats := appList(1, 1, []string{})

	res := &QASResult{
		Count:   len(stats),
		Message: "success",
		Data: &QASData{
			PlatformName:         PlatformName,
			PlatformUrl:          PlatformUrl,
			ItemNameList:         []string{"错误率", "请求延迟", "请求次数"},
			MonitorAppStatusList: make([]*MonitorAppStatus, 0),
		},
	}

	for _, stat := range stats {
		level := "绿色"
		if stat.ErrorPercent >= 10 || stat.Alive <= 0 {
			level = "红色"
		}

		res.Data.MonitorAppStatusList = append(res.Data.MonitorAppStatusList, &MonitorAppStatus{
			AppName:      stat.Name,
			WarningLevel: level,
			Items: []*MonitorAppStatusItem{
				&MonitorAppStatusItem{"错误率", strconv.FormatFloat(stat.ErrorPercent, 'f', 2, 64)},
				&MonitorAppStatusItem{"请求延迟", strconv.FormatFloat(stat.AverageElapsed, 'f', 1, 64)},
				&MonitorAppStatusItem{"请求次数", strconv.FormatInt(int64(stat.Count), 10)},
			},
		})
	}

	return c.JSON(http.StatusOK, res)
}

type QDSResult struct {
	Code    int      `json:"code"`
	Count   int      `json:"count"`
	Message string   `json:"msg"`
	Data    *QDSData `json:"data"`
}

type QDSData struct {
	SystemName   string `json:"systemName"`
	PlatformUrl  string `json:"platformUrl"`
	AppName      string `json:"applicationName"`
	WarningLevel string `json:"appWarningLevel"`

	Items []*MonitorAppStatusItem `json:"items"`
}

func QueryAppDetailStatus(c echo.Context) error {
	appName := c.FormValue("app_name")

	now := time.Now()
	end := now.Add(-2 * time.Minute)
	start := now.Add(-3 * time.Minute)

	timeline, timeBucks, suc := dashboard(appName, start, end)

	res := &QDSResult{}
	if !suc || len(timeline) == 0 {
		res.Message = "success"

		level := "绿色"
		res.Data = &QDSData{
			SystemName:   PlatformName,
			PlatformUrl:  PlatformUrl + "/apm/dashboard?app_name=" + appName,
			AppName:      appName,
			WarningLevel: level,

			Items: []*MonitorAppStatusItem{
				&MonitorAppStatusItem{"错误率", "0"},
				&MonitorAppStatusItem{"请求延迟", "0"},
				&MonitorAppStatusItem{"请求次数", "0"},
			},
		}

		return c.JSON(http.StatusOK, res)
	}

	stat := timeBucks[timeline[0]]

	res.Message = "success"

	level := "绿色"
	if stat.ErrorPercent >= 10 || stat.Alive <= 0 {
		level = "红色"
	}
	res.Data = &QDSData{
		SystemName:   PlatformName,
		PlatformUrl:  PlatformUrl + "/apm/dashboard?app_name=" + appName,
		AppName:      appName,
		WarningLevel: level,

		Items: []*MonitorAppStatusItem{
			&MonitorAppStatusItem{"错误率", strconv.FormatFloat(stat.ErrorPercent, 'f', 2, 64)},
			&MonitorAppStatusItem{"请求延迟", strconv.FormatFloat(stat.AverageElapsed, 'f', 1, 64)},
			&MonitorAppStatusItem{"请求次数", strconv.FormatInt(int64(stat.Count), 10)},
		},
	}

	return c.JSON(http.StatusOK, res)
}
