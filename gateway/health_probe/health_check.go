package health_probe

import (
	"Hamburger/gateway/runtime"
	"Hamburger/internal/config"
	"Hamburger/internal/job_syncer"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// 根据域名检查状态

func InitProbeSyncer(cfg *config.Config, logger *zerolog.Logger) {
	job_syncer.NewJobSyncer(logger,
		"sync health-probe",
		cfg.Syncer.JobSyncHealthProbe.Get(60),
		func() {
			CheckDomainHealth()
		}).Start()
}

func CheckDomainHealth() {
	// 仅检查端口组的第一个端口
	runtime.ServicePortsMap.Range(func(key string, value []int) bool {
		if len(value) == 0 {
			return true
		}
		port := value[0]
		status := netChecker(fmt.Sprintf("%s:%d", "127.0.0.1", port))
		SetProbe(key, status)
		return true
	})
}

func netChecker(addr string) []byte {
	c, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return HealthStatusDead
	}
	defer c.Close()
	return HealthStatusLive
}

// ProbeResult is one manual health check for a configured domain.
type ProbeResult struct {
	Domain    string
	Service   string
	Status    string
	CheckedAt time.Time
}

// ProbeDomain dials the domain's configured target and stores the result under
// both the service name and the domain, which are the keys the management
// list reads.
func ProbeDomain(domain string, serviceName string) ProbeResult {
	normalized := strings.ToLower(strings.TrimSpace(domain))
	status := dialProbe(probeTargets(normalized, serviceName))
	SetProbe(normalized, status)
	if serviceName != "" && !strings.EqualFold(serviceName, normalized) {
		SetProbe(serviceName, status)
	}
	checkedAt := time.Now().UTC()
	if probedAt, ok := GetProbeTime(normalized); ok {
		checkedAt = probedAt
	}
	return ProbeResult{Domain: normalized, Service: serviceName, Status: probeStatusName(status), CheckedAt: checkedAt}
}

func probeStatusName(status []byte) string {
	if string(status) == string(HealthStatusDead) {
		return "warning"
	}
	return "online"
}

func probeTargets(domain string, serviceName string) []string {
	var targets []string
	if ports, ok := runtime.ServicePortsMap.Get(serviceName); ok && len(ports) > 0 && ports[0] > 0 {
		targets = append(targets, net.JoinHostPort("127.0.0.1", strconv.Itoa(ports[0])))
	}
	if service, ok := runtime.GetService(serviceName); ok {
		if service.Host != "" && service.Port > 0 {
			targets = append(targets, net.JoinHostPort(service.Host, strconv.Itoa(service.Port)))
		}
		for _, proxy := range service.ProxyPass {
			if proxy.ProxyDirect.ProxyHost != "" && proxy.ProxyDirect.ProxyPort > 0 {
				targets = append(targets, net.JoinHostPort(proxy.ProxyDirect.ProxyHost, strconv.Itoa(proxy.ProxyDirect.ProxyPort)))
			}
		}
	}
	if len(targets) == 0 && domain != "" {
		targets = append(targets, net.JoinHostPort(domain, "443"), net.JoinHostPort(domain, "80"))
	}
	return targets
}

func dialProbe(addresses []string) []byte {
	for _, address := range addresses {
		if string(netChecker(address)) == string(HealthStatusLive) {
			return HealthStatusLive
		}
	}
	return HealthStatusDead
}
