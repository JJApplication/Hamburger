package initialize

import (
	"Hamburger/gateway/api/service"
)

// InitAPIService creates the shared API business service before either the
// standalone API listener or the gateway Connect facade is initialized.
func (i *Initializer) InitAPIService() Runner {
	return Runner{
		Priority: PriorityHigh,
		fn: func() error {
			if i.cfg == nil {
				return nil
			}
			// Keep one service instance for the process even when both HTTP
			// facades start disabled. A later in-place config reload can enable
			// Connect without needing to replace the handler or reopen the store.
			i.APIService = service.NewAPIService(i.cfg.ApiServerConfig)
			return nil
		},
	}
}
