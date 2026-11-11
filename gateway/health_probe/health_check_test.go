package health_probe

import (
	"net"
	"testing"

	"Hamburger/gateway/runtime"
)

func TestHealthCheck(t *testing.T) {
	data := netChecker("127.0.0.1:22")
	t.Log(string(data))
}

func TestDialProbeUsesFirstOpenAddress(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			conn.Close()
		}
	}()

	if string(dialProbe([]string{"127.0.0.1:1", listener.Addr().String()})) != string(HealthStatusLive) {
		t.Fatal("expected the open address to be live")
	}
	if string(dialProbe([]string{"127.0.0.1:1"})) != string(HealthStatusDead) {
		t.Fatal("expected a closed address to be dead")
	}
}

func TestProbeDomainRemembersDomainAndService(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}
			conn.Close()
		}
	}()
	port := listener.Addr().(*net.TCPAddr).Port
	const serviceName = "probe-domain-test"
	runtime.ServicePortsMap.Put(serviceName, []int{port})
	defer runtime.ServicePortsMap.Delete(serviceName)

	result := ProbeDomain("probe.example", serviceName)
	if result.Status != "online" {
		t.Fatalf("status = %s", result.Status)
	}
	if string(GetProbe("probe.example")) != string(HealthStatusLive) || string(GetProbe(serviceName)) != string(HealthStatusLive) {
		t.Fatal("probe result was not stored for the domain and service")
	}
}
