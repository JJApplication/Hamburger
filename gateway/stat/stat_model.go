package stat

type GeoModel struct {
	ID      int64  `json:"id" gorm:"column:id;primary_key"`
	ISOCode string `json:"iso_code" gorm:"column:iso_code"`
	Count   int64  `json:"count" gorm:"column:count"`
}

type StatModel struct {
	ID     int64 `json:"id" gorm:"column:id;primary_key"`
	Total  int64 `json:"total" gorm:"column:total"`
	API    int64 `json:"api" gorm:"column:api"`
	Static int64 `json:"static" gorm:"column:static"`
	Fail   int64 `json:"fail" gorm:"column:fail"`
}

type DomainModel struct {
	ID     int64  `json:"id" gorm:"column:id;primary_key"`
	Domain string `json:"domain" gorm:"column:domain"`
	Count  int64  `json:"count" gorm:"column:count"`
}
