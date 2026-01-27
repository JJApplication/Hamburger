package initialize

import "Hamburger/internal/data"

func (i *Initializer) InitMongo() Runner {
	return Runner{
		Priority: PriorityHigh,
		fn: func() error {
			data.InitMongo()
			return nil
		},
	}
}
