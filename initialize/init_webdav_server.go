package initialize

import (
	exp_webdav "Hamburger/exp/webdav"
)

func (i *Initializer) InitWebDAVServer() Runner {
	return Runner{
		Priority: PriorityLow,
		fn: func() error {
			i.WebDAVServer = exp_webdav.NewWebDAVServer(i.cfg, i.logger)
			return nil
		},
	}
}
