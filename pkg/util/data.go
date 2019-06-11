package util

/* 公共数据结构 */

type Alert struct {
	Name     string  `json:"name" cql:"name"`
	Type     string  `json:"type" cql:"type"`
	Label    string  `json:"label" cql:"label"`
	Compare  int     `json:"compare" cql:"compare"`
	Unit     string  `json:"unit" cql:"unit"`
	Duration int     `json:"duration" cql:"duration"`
	Keys     string  `json:"keys" cql:"keys"`
	Value    float64 `json:"value" cql:"value"`
}

// ApiAlert ...
type ApiAlert struct {
	Api    string     `json:"api" cql:"api"`
	Alerts []*AlertKV `json:"alerts" cql:"alerts"`
}

type AlertKV struct {
	Key   string  `json:"key" cql:"key"`
	Value float64 `json:"value" cql:"value"`
}
