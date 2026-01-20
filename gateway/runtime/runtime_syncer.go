package runtime

import (
	"Hamburger/internal/config"
	"Hamburger/internal/job_syncer"
	"github.com/rs/zerolog"
)

func InitRuntimeSyncer(cfg *config.AppConfig, logger *zerolog.Logger) {
	job_syncer.NewJobSyncer(logger,
		"sync runtime-domains",
		cfg.Syncer.JobSyncDomainsMap.Get(3600),
		func() {
			loadRuntimeDomains(cfg)
		}).Start()

	job_syncer.NewJobSyncer(logger,
		"sync runtime-domain-ports",
		cfg.Syncer.JobSyncDomainPorts.Get(3600),
		func() {
			RefreshDomainPortsMap()
		}).Start()
}
