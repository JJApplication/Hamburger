package health_probe

import (
	"Hamburger/gateway/runtime"
	"Hamburger/internal/config"
	"Hamburger/internal/job_syncer"
	"fmt"
	"net"
	"time"

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
	runtime.ServicePortsMap.Range(func(key string, value []int) bool {
		if len(value) == 0 {
			return true
		}
		port := value[0]
		status := netChecker(fmt.Sprintf("%s:%d", "127.0.0.1", port))
		SetProbe(key, status)
		return true
	})
}

func netChecker(addr string) []byte {
	c, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return HealthStatusDead
	}
	defer c.Close()
	return HealthStatusLive
}
