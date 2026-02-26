package statement_route

import (
	"Hamburger/internal/json"
	"os"
	"sync"
)

// 每个服务以单文件的方式声明路由

type StateRoute struct {
	StateType string   `json:"state_type"` // 前端 后端
	ID        string   `json:"id"`
	Tags      []string `json:"tags"`    // 标签
	Service   string   `json:"service"` // 服务名称
	Http      struct {
		Domain    string   `json:"domain"`  // 服务关联的域名
		Target    []string `json:"target"`  // ip:port
		Timeout   int      `json:"timeout"` // 超时控制
		ProxyPass []struct {
			Timeout      int      `json:"timeout"`
			Path         string   `json:"path"`
			AllowMethods []string `json:"allow_methods"`
			Target       string   `json:"target"`
			RewritePath  string   `json:"rewrite_path"`
		} `json:"proxy_pass"` // 代理转发后端
	} `json:"http"` // http web服务
}

type StateRouteManager struct {
	srs   []StateRoute
	rLock sync.RWMutex
}

func NewStateRouteManager(routePath string) *StateRouteManager {
	srs, err := LoadStatementRoutes(routePath)
	if err != nil {
		return nil
	}
	return &StateRouteManager{
		srs: srs,
	}
}

func LoadStatementRoutes(routePath string) ([]StateRoute, error) {
	var srs []StateRoute
	jsonData, err := os.ReadFile(routePath)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(jsonData, &srs)
	return srs, nil
}

func (s *StateRouteManager) FindByDomain(domain string) (StateRoute, bool) {
	for _, sr := range s.srs {
		if sr.Http.Domain == domain {
			return sr, true
		}
	}

	return StateRoute{}, false
}

func (s *StateRouteManager) FindById(id string) (StateRoute, bool) {
	for _, sr := range s.srs {
		if sr.ID == id {
			return sr, true
		}
	}
	return StateRoute{}, false
}

func (s *StateRouteManager) FindByTag(tag string) []StateRoute {
	var srs []StateRoute
	for _, sr := range s.srs {
		for _, t := range sr.Tags {
			if t == tag {
				srs = append(srs, sr)
			}
		}
	}

	return srs
}

func (s *StateRouteManager) FindByService(name string) (StateRoute, bool) {
	for _, sr := range s.srs {
		if sr.Service == name {
			return sr, true
		}
	}
	return StateRoute{}, false
}
