package runtime

import (
	"Hamburger/internal/structure"
	"fmt"
)

// 域名和后端端口映射
type GetDomainPortsMap interface {
	LoadDomainPorts() *structure.Map[[]int]
}

var (
	DomainPortsMap *structure.Map[[]int]
)

func init() {
	DomainPortsMap = structure.NewMap[[]int]()
}

// InitDomainPortsMap 初始化域名端口组映射
func InitDomainPortsMap() {
	loadDomainPortsMap()
}

// RefreshDomainPortsMap 更新端口组
func RefreshDomainPortsMap() {
	loadDomainPortsMap()
}

func loadDomainPortsMap() {
	DomainPortsMap = (&DomainPortsMongoAdapter{}).LoadDomainPorts()
}

func getDomainPort(host string) []int {
	if d, ok := DomainPortsMap.Get(host); ok {
		return d
	}
	return nil
}

// DomainReflect 将端口转换为ip地址 单机的ip都是127.0.0.1
func DomainReflect(host string) []string {
	group := getDomainPort(host)
	if len(group) == 0 {
		return nil
	}
	var dGroup []string
	for _, v := range group {
		dGroup = append(dGroup, fmt.Sprintf("127.0.0.1:%d", v))
	}

	return dGroup
}

func GetDomainPortsSnapshot() map[string][]int {
	result := map[string][]int{}
	DomainPortsMap.Range(func(key string, value []int) bool {
		ports := make([]int, len(value))
		copy(ports, value)
		result[key] = ports
		return true
	})
	return result
}
