package initialize

import "Hamburger/gateway/stat"

func (i *Initializer) InitStatServer() Runner {
	return Runner{
		Priority: PriorityLow,
		fn: func() error {
			i.StatServer = stat.NewStatServer(i.cfg.Stat, i.logger)
			return nil
		}}
}
