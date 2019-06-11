package util

import (
	"io/ioutil"
	"os"
	"strings"
)

type AgentdInfo struct {
	Hostname         string `json:"hn"`
	StartTime        int64  `json:"st"`
	DownloadTimes    int    `json:"dt"`
	RestartTimes     int    `json:"rt"`
	LastDownloadTime int64  `json:"ldt"`
	LastRestartTime  int64  `json:"lrt"`
	AgentdVersion    string `json:"adv"`
	AgentVersion     string `json:"av"`
	DownloadVersion  string `json:"dv"`
	AdminPort        string `json:"ap"`
	UpdateTime       int64  `json:"ut"`
}

type HealthResult struct {
	Success bool   `json:"sc"`
	Version string `json:"v"`
}

func GetVersion(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	v, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(v)), nil
}
