package runtime

import (
	"Hamburger/internal/structure"
	"fmt"
)

// GetDomainPortsMap 服务和后端端口映射
type GetDomainPortsMap interface {
	LoadServicePorts() *structure.Map[[]int]
}

var (
	ServicePortsMap *structure.Map[[]int]
)

func init() {
	ServicePortsMap = structure.NewMap[[]int]()
}

// InitServicePortsMap 初始化域名端口组映射
func InitServicePortsMap() {
	loadServicePortsMap()
}

// RefreshServicePortsMap 更新端口组
func RefreshServicePortsMap() {
	loadServicePortsMap()
}

func loadServicePortsMap() {
	ServicePortsMap = (&ServicePortsMongoAdapter{}).LoadServicePorts()
}

func getServicePort(service string) []int {
	if d, ok := ServicePortsMap.Get(service); ok {
		return d
	}
	return nil
}

// ServiceReflect 将端口转换为ip地址 单机的ip都是127.0.0.1
func ServiceReflect(service string) []string {
	group := getServicePort(service)
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
	ServicePortsMap.Range(func(key string, value []int) bool {
		ports := make([]int, len(value))
		copy(ports, value)
		result[key] = ports
		return true
	})
	return result
}
