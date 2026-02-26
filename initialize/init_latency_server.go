package initialize

import "Hamburger/gateway/latency"

func (i *Initializer) InitLatencyServer() Runner {
	return Runner{
		Priority: PriorityLow,
		fn: func() error {
			i.LatencyServer = latency.NewLatencyServer(i.cfg.Latency, i.logger)
			return nil
		}}
}
