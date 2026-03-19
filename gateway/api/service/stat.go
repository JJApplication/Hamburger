package service

import "Hamburger/gateway/stat"

func (s *APIService) GetStatCounters() map[string]int64 {
	return map[string]int64{
		"total":  stat.Get(stat.Total),
		"api":    stat.Get(stat.API),
		"static": stat.Get(stat.Static),
		"fail":   stat.Get(stat.Fail),
		"today":  stat.Get(stat.Today),
	}
}

func (s *APIService) GetGeoData() []byte {
	return stat.GetGeoData()
}

func (s *APIService) GetDomainData() []byte {
	return stat.GetDomainStat()
}

func (s *APIService) GetConnData() interface{} {
	var connData = map[string]interface{}{}
	connGw := stat.GetGatewayConn()
	connFront := stat.GetFrontConn()
	connData["gateway"] = connGw
	connData["front"] = connFront

	return connData
}
