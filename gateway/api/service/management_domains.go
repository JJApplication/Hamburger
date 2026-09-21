package service

import (
	"Hamburger/gateway/health_probe"
	"Hamburger/gateway/prehandler"
	"Hamburger/gateway/runtime"
	"Hamburger/gateway/stat"
	"Hamburger/internal/config"
	"Hamburger/internal/config/loader"
	"net"
	"strconv"
	"strings"
	"time"
)

// ManagementDomainMapping describes one configured listener-to-backend route.
// It is shared by the REST and Connect management facades.
type ManagementDomainMapping struct {
	FrontendPort  int    `json:"frontend_port"`
	BackendTarget string `json:"backend_target"`
	Protocol      string `json:"protocol"`
}

// ManagementDomain is the runtime view used by the dashboard.  Configured
// domains are returned even when they have not received a request yet.
type ManagementDomain struct {
	Domain             string                    `json:"domain"`
	Service            string                    `json:"service"`
	Status             string                    `json:"status"`
	LastProbe          string                    `json:"last_probe"`
	CurrentConnections int64                     `json:"current_connections"`
	PeakConnections    int64                     `json:"peak_connections"`
	Mappings           []ManagementDomainMapping `json:"mappings"`
}

// GetManagementDomains builds one bounded snapshot from the configured
// service rules, listener ports, probe cache, and connection tracker.  The
// same method is deliberately used by REST and Connect so the two protocols
// cannot drift in fields or status semantics.
func (s *APIService) GetManagementDomains() []ManagementDomain {
	domains, serviceMap, _ := runtime.GetDomainsSnapshot()
	ports := runtime.GetDomainPortsSnapshot()
	connections := stat.GetDomainConnSnapshot()
	manager := prehandler.GetServiceManager()
	var cfg *config.Config
	if s != nil && s.cfg != nil {
		cfg = loader.SnapshotOf(s.cfg)
	}
	items := make([]ManagementDomain, 0, len(domains))
	for _, domain := range domains {
		serviceName := serviceMap[domain]
		if serviceName == "" {
			serviceName = serviceMap[strings.ToLower(strings.TrimSpace(domain))]
		}
		item, found := runtime.GetService(serviceName)
		mappings := make([]ManagementDomainMapping, 0)
		if found {
			entryPort := item.Port
			if entryPort == 0 && len(ports[serviceName]) > 0 {
				entryPort = ports[serviceName][0]
			}
			for _, proxy := range item.ProxyPass {
				target := proxy.ProxyDirect.ProxyHost
				if target == "" {
					target = item.Host
					if target == "" && cfg != nil && strings.EqualFold(item.ServiceType, "frontend") {
						target = cfg.PxyFrontend.Host
					}
				}
				port := proxy.ProxyDirect.ProxyPort
				if port == 0 {
					port = item.Port
					if port == 0 && cfg != nil && strings.EqualFold(item.ServiceType, "frontend") {
						port = cfg.PxyFrontend.Port
					}
				}
				mappings = append(mappings, ManagementDomainMapping{
					FrontendPort:  entryPort,
					BackendTarget: managementFormatTarget(target, port),
					Protocol:      managementProtocol(cfg, domain, entryPort, item.ServiceType),
				})
			}
			if len(mappings) == 0 {
				for _, port := range ports[serviceName] {
					target := item.Host
					if target == "" && cfg != nil && strings.EqualFold(item.ServiceType, "frontend") {
						target = cfg.PxyFrontend.Host
					}
					mappings = append(mappings, ManagementDomainMapping{
						FrontendPort:  port,
						BackendTarget: managementFormatTarget(target, 0),
						Protocol:      managementProtocol(cfg, domain, port, item.ServiceType),
					})
				}
			}
		}

		// Health probes are keyed by the runtime service name (the probe
		// checker tests its first listener), while older deployments may have
		// stored a domain key.  Prefer the service key and keep the fallback for
		// compatibility.
		probeKey := serviceName
		probe := health_probe.GetProbe(probeKey)
		if len(probe) == 0 {
			probeKey = domain
			probe = health_probe.GetProbe(probeKey)
		}
		state := "unknown"
		if manager.IsDomainStopped(domain) {
			state = "offline"
		} else if len(probe) == 0 {
			// Unknown is distinct from a healthy zero-value.  A configured
			// domain may not have been probed since process start.
			state = "unknown"
		} else if string(probe) == string(health_probe.HealthStatusDead) {
			state = "warning"
		} else {
			state = "online"
		}
		lastProbe := ""
		if probeTime, ok := health_probe.GetProbeTime(probeKey); ok {
			lastProbe = probeTime.Format(time.RFC3339)
		}
		counts := connections[domain]
		if counts == nil {
			counts = connections[strings.ToLower(strings.TrimSpace(domain))]
		}
		items = append(items, ManagementDomain{
			Domain:             domain,
			Service:            serviceName,
			Status:             state,
			LastProbe:          lastProbe,
			CurrentConnections: counts["active"] + counts["idle"],
			PeakConnections:    counts["peak"],
			Mappings:           mappings,
		})
	}
	return items
}

func managementFormatTarget(host string, port int) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return "未知"
	}
	if port <= 0 {
		return host
	}
	if _, _, err := net.SplitHostPort(host); err == nil {
		return host
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}

func managementProtocol(cfg *config.Config, domain string, port int, fallback string) string {
	protocol := strings.ToLower(strings.TrimSpace(fallback))
	if cfg != nil {
		normalized := strings.ToLower(strings.TrimSpace(domain))
		for _, server := range cfg.Servers {
			match := port > 0 && server.Port == port
			for _, binding := range server.DomainConfig {
				for _, name := range append(append([]string{}, binding.Domains...), binding.WsDomains...) {
					if strings.EqualFold(strings.TrimSpace(name), normalized) {
						match = true
					}
				}
			}
			if match && strings.TrimSpace(server.Protocol) != "" {
				protocol = strings.ToLower(strings.TrimSpace(server.Protocol))
				break
			}
		}
	}
	if protocol == "frontend" || protocol == "backend" || protocol == "custom" || protocol == "" {
		return "http"
	}
	return protocol
}
