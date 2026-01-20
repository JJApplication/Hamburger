package geo

import (
	"Hamburger/internal/logger"
	_ "embed"
	"github.com/oschwald/maxminddb-golang/v2"
	"net/netip"
)

// 处理请求与地理位置的映射关系
// https://github.com/P3TERX/GeoLite.mmdb/releases

//go:embed GeoLite2-Country.mmdb
var geoData []byte
var db *maxminddb.Reader

func LoadGEO() {
	mmdb, err := maxminddb.OpenBytes(geoData)
	if err != nil {
		logger.GetLogger().Error().Err(err).Msg("maxminddb.Open err")
		return
	}
	db = mmdb
}

func GeoLookUp(ip string) string {
	if db == nil {
		return ""
	}
	ipAddr, err := netip.ParseAddr(ip)
	if err != nil {
		return ""
	}
	var record struct {
		Country struct {
			ISOCode string            `maxminddb:"iso_code"`
			Names   map[string]string `maxminddb:"names"`
		} `maxminddb:"country"`
	}

	if err = db.Lookup(ipAddr).Decode(&record); err != nil {
		return ""
	}

	return record.Country.ISOCode
}
