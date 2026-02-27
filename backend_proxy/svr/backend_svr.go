package svr

type BackendSvr interface {
	Start()
	Stop() error
	IsStarted() bool
	Name() string
	GetAddr() string
}
