package health_probe

import (
	"github.com/lesismal/nbio"
	"github.com/lesismal/nbio/logging"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	logging.SetLogger(nil)
	logging.SetLevel(logging.LevelNone)
	g := nbio.NewGopher(nbio.Config{})
	g.Start()
	defer g.Stop()
	data := nbioChecker(g, "127.0.0.1:22")
	t.Log(string(data))
}
