package initialize

import (
	"Hamburger/gateway/geo"
	"Hamburger/gateway/stat"
)
import "Hamburger/gateway/stat/db"

// 初始化状态记录器

func (i *Initializer) InitStatManager() Runner {
	return Runner{
		Priority: PriorityLow,
		fn: func() error {
			if err := db.Init(i.cfg); err != nil {
				return err
			}
			// load GEO
			geo.LoadGEO()
			// init syncer
			stat.InitStatSyncer()
			return nil
		}}
}
