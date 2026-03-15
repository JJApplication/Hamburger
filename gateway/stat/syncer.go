package stat

import (
	"time"

	"Hamburger/internal/logger"
)

func (m *StatManager) InitStatSyncer() {
	cfg := m.getCfg()
	if !cfg.Stat.EnableStat {
		return
	}
	m.initCacheFromFile()
	du := cfg.Stat.SyncDuration
	if du == 0 {
		du = 60
	}

	sdu := cfg.Stat.SaveDuration
	if sdu == 0 {
		sdu = 3600
	}

	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(du))
		defer ticker.Stop()
		for range ticker.C {
			logger.GetLogger().Info().Msg("running stat syncer")
			go m.syncStat()
			go m.syncGEOStat()
			go m.syncDomainStat()
		}
	}()

	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(sdu))
		defer ticker.Stop()
		for range ticker.C {
			logger.GetLogger().Info().Msg("save stat to file")
			go m.SaveStat()
			go m.SaveGeoStat()
			go m.SaveDomainStat()
		}
	}()
}
