package health_probe

import "testing"

func TestHealthCheck(t *testing.T) {
	data := netChecker("127.0.0.1:22")
	t.Log(string(data))
}
