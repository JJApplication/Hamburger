package job_syncer

import (
	"github.com/rs/zerolog"
	"time"
)

// 定时任务触发器

type JobSyncer struct {
	TaskName string
	SyncTime time.Duration
	fn       func()
	logger   *zerolog.Logger
	ticker   *time.Ticker
}

func NewJobSyncer(logger *zerolog.Logger, taskName string, syncTime time.Duration, fn func()) *JobSyncer {
	if syncTime <= 0 {
		syncTime = 60 * time.Second // 设置最小时间
	}
	ticker := time.NewTicker(syncTime)
	return &JobSyncer{
		TaskName: taskName,
		SyncTime: syncTime,
		fn:       fn,
		logger:   logger,
		ticker:   ticker,
	}
}

func (j *JobSyncer) Start() {
	go func() {
		for {
			select {
			case <-j.ticker.C:
				j.logger.Info().Str("task", j.TaskName).Msg("job sync start")
				j.fn()
				j.logger.Info().Str("task", j.TaskName).Msg("job sync finished")
			}
		}
	}()
}

func (j *JobSyncer) Stop() {
	j.ticker.Stop()
}
