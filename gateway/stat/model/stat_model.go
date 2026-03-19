package model

// GeoModel geo数据
type GeoModel struct {
	ID      int64  `json:"id" gorm:"column:id;primary_key"`
	ISOCode string `json:"iso_code" gorm:"column:iso_code"`
	Count   int64  `json:"count" gorm:"column:count"`
}

// StatModel 请求数据
type StatModel struct {
	ID     int64 `json:"id" gorm:"column:id;primary_key"`
	Total  int64 `json:"total" gorm:"column:total"`
	API    int64 `json:"api" gorm:"column:api"`
	Static int64 `json:"static" gorm:"column:static"`
	Fail   int64 `json:"fail" gorm:"column:fail"`
}

// DomainModel 域名请求数据
type DomainModel struct {
	ID     int64  `json:"id" gorm:"column:id;primary_key"`
	Domain string `json:"domain" gorm:"column:domain"`
	Count  int64  `json:"count" gorm:"column:count"`
}

// GatewayConnModel 网关代理连接数
//
// 当前没有存储必要 是每次启动后的临时数据
type GatewayConnModel struct {
	New      int64 `json:"new" gorm:"column:new"`
	Active   int64 `json:"active" gorm:"column:active"`
	Idle     int64 `json:"idle" gorm:"column:idle"`
	Hijacked int64 `json:"hijacked" gorm:"column:hijacked"`
	Closed   int64 `json:"closed" gorm:"column:closed"`
}

// FrontConnModel 前端服务连接数
type FrontConnModel struct {
	New      int64 `json:"new" gorm:"column:new"`
	Active   int64 `json:"active" gorm:"column:active"`
	Idle     int64 `json:"idle" gorm:"column:idle"`
	Hijacked int64 `json:"hijacked" gorm:"column:hijacked"`
	Closed   int64 `json:"closed" gorm:"column:closed"`
}
