package initialize

import (
	"Hamburger/gateway/notifier"
	"Hamburger/internal/queue"
)

// InitNotifier 初始化通知服务
func (i *Initializer) InitNotifier() Runner {
	return Runner{
		Priority: PriorityLow,
		fn: func() error {
			if !i.cfg.NotifyConfig.Enabled {
				return nil
			}
			queue.InitDefault(i.cfg.NotifyConfig.Queue.Topic, i.cfg.NotifyConfig.Queue.Buffer)
			service, err := notifier.Register(i.cfg.NotifyConfig)
			if err != nil {
				return err
			}
			i.Notifier = service
			return nil
		},
	}
}
