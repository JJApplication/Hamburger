package health_probe

import (
	"Hamburger/gateway/runtime"
	"Hamburger/internal/config"
	"Hamburger/internal/job_syncer"
	"fmt"
	"github.com/lesismal/nbio"
	"github.com/lesismal/nbio/logging"
	"github.com/rs/zerolog"
)

// 根据域名检查状态

func InitProbeSyncer(cfg *config.Config, logger *zerolog.Logger) {
	job_syncer.NewJobSyncer(logger,
		"sync health-probe",
		cfg.Syncer.JobSyncHealthProbe.Get(60),
		func() {
			CheckDomainHealth()
		}).Start()
}

func CheckDomainHealth() {
	// 仅检查端口组的第一个端口
	logging.SetLogger(nil)
	logging.SetLevel(logging.LevelNone)
	g := nbio.NewGopher(nbio.Config{})
	g.Start()
	defer g.Stop()
	runtime.DomainPortsMap.Range(func(key string, value []int) bool {
		if len(value) == 0 {
			return true
		}
		port := value[0]
		status := nbioChecker(g, fmt.Sprintf("%s:%d", "127.0.0.1", port))
		SetProbe(key, status)
		return true
	})
}

func nbioChecker(g *nbio.Gopher, addr string) []byte {
	c, err := nbio.Dial("tcp", addr)
	if err != nil {
		return HealthStatusDead
	}
	g.AddConn(c)

	defer c.Close()
	return HealthStatusLive
}
