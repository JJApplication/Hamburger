package stat

import (
	"Hamburger/internal/config"
	"Hamburger/internal/logger"
	"time"
)

func InitStatSyncer() {
	cfg := config.Get()
	if !cfg.Stat.EnableStat {
		return
	}
	initCacheFromFile()
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
			go syncStat()
			go syncGEOStat()
			go syncDomainStat()
		}
	}()

	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(sdu))
		defer ticker.Stop()
		for range ticker.C {
			logger.GetLogger().Info().Msg("save stat to file")
			go SaveStat(cfg)
			go SaveGeoStat()
			go SaveDomainStat()
		}
	}()
}
