package stat

import (
	"context"
	"time"

	"Hamburger/gateway/stat/db"
	"Hamburger/internal/config/svr_config"
	"Hamburger/internal/logger"
)

func (m *StatManager) InitStatSyncer() {
	m.syncOnce.Do(func() {
		cfg := m.getCfg()
		if cfg == nil {
			return
		}
		legacyEnabled := cfg.Stat.EnableStat
		historyEnabled := cfg.Stat.Sequence.Enabled && db.GetDB() != nil
		if !legacyEnabled && !historyEnabled {
			return
		}
		if legacyEnabled {
			m.initCacheFromFile()
		}

		if historyEnabled {
			store, err := NewHistoryStore(db.GetDB(), cfg.Stat.Sequence)
			if err != nil {
				logger.GetLogger().Error().Err(err).Msg("initialize stat history store failed")
			} else {
				m.history = store
				m.sampler = newResourceSampler()
			}
		}

		du := cfg.Stat.SyncDuration
		if du <= 0 {
			du = 60
		}
		sdu := cfg.Stat.SaveDuration
		if sdu <= 0 {
			sdu = 3600
		}
		flushSeconds, cleanupSeconds := sequenceIntervals(cfg.Stat.Sequence)
		ctx, cancel := context.WithCancel(context.Background())
		m.syncCancel = cancel

		if legacyEnabled {
			m.syncWG.Add(1)
			go func() {
				defer m.syncWG.Done()
				ticker := time.NewTicker(time.Second * time.Duration(du))
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						logger.GetLogger().Info().Msg("running stat syncer")
						m.syncStat()
						m.syncGEOStat()
						m.syncDomainStat()
					case <-ctx.Done():
						return
					}
				}
			}()

			m.syncWG.Add(1)
			go func() {
				defer m.syncWG.Done()
				ticker := time.NewTicker(time.Second * time.Duration(sdu))
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						logger.GetLogger().Info().Msg("save stat to file")
						m.SaveStat()
						m.SaveGeoStat()
						m.SaveDomainStat()
					case <-ctx.Done():
						return
					}
				}
			}()
		}

		if m.history != nil {
			m.syncWG.Add(1)
			go m.runHistorySyncer(ctx, flushSeconds, cleanupSeconds)
		}
	})
}

func sequenceIntervals(cfg svr_config.SequenceConfig) (int, int) {
	flushSeconds := cfg.FlushInterval
	if flushSeconds <= 0 {
		flushSeconds = 5
	}
	cleanupSeconds := cfg.CleanupInterval
	if cleanupSeconds <= 0 {
		cleanupSeconds = 3600
	}
	return flushSeconds, cleanupSeconds
}

func (m *StatManager) runHistorySyncer(ctx context.Context, flushSeconds, cleanupSeconds int) {
	defer m.syncWG.Done()
	flushTicker := time.NewTicker(time.Duration(flushSeconds) * time.Second)
	defer flushTicker.Stop()
	cleanupTicker := time.NewTicker(time.Duration(cleanupSeconds) * time.Second)
	defer cleanupTicker.Stop()
	for {
		select {
		case now := <-flushTicker.C:
			if m.sampler != nil {
				m.history.RecordResource(m.sampler.sample(ctx, now.UTC()))
			}
			if err := m.history.Flush(); err != nil {
				logger.GetLogger().Error().Err(err).Msg("flush stat history failed")
			}
		case now := <-cleanupTicker.C:
			if err := m.history.CleanupExpired(now.UTC()); err != nil {
				logger.GetLogger().Error().Err(err).Msg("cleanup stat history failed")
			}
		case <-ctx.Done():
			return
		}
	}
}

// HistoryStore returns the active fixed-table historical store, if enabled.
func (m *StatManager) HistoryStore() *HistoryStore {
	if m == nil {
		return nil
	}
	return m.history
}

// Close stops collectors, forces the final historical batch to SQLite, then
// closes the shared database. Legacy cumulative data is flushed first.
func (m *StatManager) Close() error {
	if m == nil {
		return nil
	}
	m.closeOnce.Do(func() {
		if m.syncCancel != nil {
			m.syncCancel()
			m.syncWG.Wait()
		}
		cfg := m.getCfg()
		if cfg != nil && cfg.Stat.EnableStat {
			m.syncStat()
			m.syncGEOStat()
			m.syncDomainStat()
			m.SaveStat()
			m.SaveGeoStat()
			m.SaveDomainStat()
		}
		if m.history != nil {
			m.closeErr = m.history.Close()
		}
		if err := db.Close(); m.closeErr == nil {
			m.closeErr = err
		}
	})
	return m.closeErr
}
