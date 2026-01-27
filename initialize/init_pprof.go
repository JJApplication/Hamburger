package initialize

import "Hamburger/internal/pprof"

func (i *Initializer) InitPProf() Runner {
	return Runner{
		Priority: PriorityLow,
		fn: func() error {
			pprof.InitPProf(struct {
				Enable bool `yaml:"enable" json:"enable"`
				Port   int  `yaml:"port" json:"port"`
			}{Enable: i.cfg.PProf.Enable, Port: i.cfg.PProf.Port})
			i.logger.Info().Msg("init pprof success")
			return nil
		},
	}
}
