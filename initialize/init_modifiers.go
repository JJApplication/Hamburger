package initialize

import "Hamburger/gateway/modifier"

func (i *Initializer) InitModifierManager() Runner {
	return Runner{
		Priority: PriorityLow,
		fn: func() error {
			modifier.InitModifiers()
			return nil
		},
	}
}
