package runtime

import (
	"Hamburger/internal/config"
	"Hamburger/internal/structure"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

func TestLoadRuntimeDomainsIndexesExactAndRegexSeparately(t *testing.T) {
	path := filepath.Join(t.TempDir(), "domains.json")
	data := `{"services":[
{"service_name":"ExactFront","service_type":"frontend","service_domain":"example.com"},
{"service_name":"RegexFront","service_type":"frontend","service_domain":"/^api-[a-z]+\\.example\\.com$/"}
]}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	loadRuntimeDomains(&config.AppConfig{DomainMap: path})

	if got := len(DomainsRuntimeMap.RegexDomains); got != 1 {
		t.Fatalf("regex domain count = %d, want 1", got)
	}
	if _, ok := DomainsRuntimeMap.DomainsMap.Get("example.com"); !ok {
		t.Fatal("ordinary domain was not pre-indexed")
	}
	service, ok := GetDomain2Service("EXAMPLE.COM:443")
	if !ok || service.ServiceName != "ExactFront" {
		t.Fatalf("exact domain lookup = %#v, %v", service, ok)
	}
	service, ok = GetDomain2Service("api-one.example.com")
	if !ok || service.ServiceName != "RegexFront" {
		t.Fatalf("regex domain lookup = %#v, %v", service, ok)
	}
	if _, ok := DomainsRuntimeMap.DomainsMap.Get("api-one.example.com"); !ok {
		t.Fatal("regex match was not cached by normalized host")
	}
}

func BenchmarkGetDomain2ServiceExact(b *testing.B) {
	service := config.Service{ServiceName: "ExactFront", ServiceType: "frontend", ServiceDomain: "example.com"}
	DomainLock.Lock()
	DomainsRuntimeMap = struct {
		Domains        []string
		DomainsMap     *structure.Map[config.Service]
		RegexDomains   []RegexDomainService
		DomainFrontMap *structure.Map[string]
		ServiceMap     *structure.Map[config.Service]
	}{
		DomainsMap:     structure.NewMap[config.Service](),
		DomainFrontMap: structure.NewMap[string](),
		ServiceMap:     structure.NewMap[config.Service](),
	}
	DomainsRuntimeMap.DomainsMap.Put("example.com", service)
	DomainsRuntimeMap.ServiceMap.Put(service.ServiceName, service)
	DomainLock.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := GetDomain2Service("example.com"); !ok {
			b.Fatal("exact domain lookup failed")
		}
	}
}

func BenchmarkGetDomain2ServiceRegex(b *testing.B) {
	service := config.Service{ServiceName: "RegexFront", ServiceType: "frontend", ServiceDomain: "/^api-[a-z0-9]+\\.example\\.com$/"}
	DomainLock.Lock()
	DomainsRuntimeMap = struct {
		Domains        []string
		DomainsMap     *structure.Map[config.Service]
		RegexDomains   []RegexDomainService
		DomainFrontMap *structure.Map[string]
		ServiceMap     *structure.Map[config.Service]
	}{
		DomainsMap:     structure.NewMap[config.Service](),
		RegexDomains:   []RegexDomainService{{Pattern: service.ServiceDomain, Service: service, Matcher: regexp.MustCompile(`^api-[a-z0-9]+\.example\.com$`)}},
		DomainFrontMap: structure.NewMap[string](),
		ServiceMap:     structure.NewMap[config.Service](),
	}
	DomainsRuntimeMap.ServiceMap.Put(service.ServiceName, service)
	DomainLock.Unlock()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		host := "api-" + strconv.Itoa(i) + ".example.com"
		if _, ok := GetDomain2Service(host); !ok {
			b.Fatal("regex domain lookup failed")
		}
		DomainsRuntimeMap.DomainsMap.Delete(host)
	}
}
