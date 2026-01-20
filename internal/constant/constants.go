//go:build v1

/*
   Create: 2025/8/5
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package constant

import (
	"time"
)

var (
	Host         string
	Port         string
	MongoUrl     string
	DBName       string
	SyncTime     time.Duration
	InfluxUrl    string
	InfluxToken  string
	InfluxOrg    string
	InfluxBucket string
	InfluxPwd    string
	EnableInflux bool
	CacheSize    int
	StrictMode   bool // 启用http严格返回模式 此情况下非2xx状态码后的响应无效
	Debug        bool
	Gzip         bool // 受网关影响不做多次gzip压缩
	// NoEngineDomain NoEngine域名和服务映射
	NoEngineDomain string // eg: blog.renj.io -> BlogFront

	HeliosAddress string
	FrontendFlag  string
	FrontendHost  string
	FrontendPort  int

	TraceID string
)

var (
	// LIMIT 限制100/10s
	LIMIT int
	// RESET 经过RESET * Duration次无请求后，从map中删除定时器
	RESET int
)

var (
	BreakerLimit int // 限制内部错误的次数
	BreakerReset int // 默认的重置时间当服务down时 等待60s后重试
)

var (
	FrontendHostHeader string // 前端服务HOST标识
	BackendHeader      string // 后端服务标识
	ProxyApp           string // 要转到的后端服务
)

func InitConfigFromEnvs() {
	Host = LoaderEnv("Host").String("127.0.0.1")
	Port = LoaderEnv("Port").String("8888")
	MongoUrl = LoaderEnv("MongoUrl").String("mongodb://localhost:27017")
	DBName = LoaderEnv("DBName").String("ApolloMongo")
	SyncTime = time.Duration(LoaderEnv("SyncTime").Int(60*5)) * time.Second
	InfluxUrl = LoaderEnv("InfluxUrl").String("http://127.0.0.1:8086")
	InfluxToken = LoaderEnv("InfluxToken").String("")
	InfluxOrg = LoaderEnv("InfluxOrg").String(Sandwich)
	InfluxBucket = LoaderEnv("InfluxBucket").String(Sandwich)
	InfluxPwd = LoaderEnv("InfluxPwd").String("")
	EnableInflux = LoaderEnv("EnableInflux").Bool(false)
	CacheSize = LoaderEnv("CacheSize").Int(10)
	StrictMode = LoaderEnv("StrictMode").Bool(false)
	Debug = LoaderEnv("Debug").Bool(false)
	Gzip = LoaderEnv("Gzip").Bool(false)
	NoEngineDomain = LoaderEnv("NoEngineDomain").String("domain.json")
	HeliosAddress = LoaderEnv("HeliosAddress").String("/var/run/Helios.sock")
	FrontendFlag = LoaderEnv("FrontendFlag").String("X-Proxy-Internal-Front")
	FrontendHost = LoaderEnv("FrontendHost").String("127.0.0.1")
	FrontendPort = LoaderEnv("FrontendPort").Int(7777)

	FrontendHostHeader = LoaderEnv("FrontendHostHeader").String("X-Proxy-Internal-Host")
	BackendHeader = LoaderEnv("BackendFlag").String("X-Proxy-Internal-Local")
	ProxyApp = LoaderEnv("ProxyApp").String("X-Proxy-Backend")

	TraceID = LoaderEnv("TraceID").String("X-Gateway-Trace-Id")

	LIMIT = LoaderEnv("FlowLimit").Int(200)
	RESET = LoaderEnv("FlowReset").Int(10)

	BreakerLimit = LoaderEnv("BreakerLimit").Int(10)
	BreakerReset = LoaderEnv("BreakerReset").Int(60)
}
