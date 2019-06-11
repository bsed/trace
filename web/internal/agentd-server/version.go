package agentd

import (
	"time"

	"github.com/imdevlab/g"
	"github.com/bsed/trace/pkg/util"
	"go.uber.org/zap"
)

var agentVersion string

func scanVersion() {
	path := "download/version"
	go func() {
		for {
			version, err := util.GetVersion(path)
			if err != nil {
				g.L.Error("agentd scan version error!", zap.Error(err), zap.String("path", path))
			}
			agentVersion = version
			time.Sleep(10 * time.Second)
		}
	}()
}
