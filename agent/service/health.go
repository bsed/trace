package service

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/imdevlab/g/utils"

	"github.com/bsed/trace/agent/misc"
	"github.com/bsed/trace/pkg/util"
	"go.uber.org/zap"
)

func initHealth() {
	go func() {
		h := http.HandlerFunc(health)
		http.Handle("/health", h)
		err := http.ListenAndServe(misc.Conf.Health.Addr, nil)
		if err != nil {
			logger.Fatal("init health check error", zap.Error(err))
		}
	}()
}

func health(w http.ResponseWriter, r *http.Request) {
	res := &util.HealthResult{
		Success: true,
		Version: misc.Conf.Common.Version,
	}
	b, _ := json.Marshal(res)
	io.WriteString(w, utils.Bytes2String(b))
}
