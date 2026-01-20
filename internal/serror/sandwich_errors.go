/*
   Create: 2024/1/22
   Project: Sandwich
   Github: https://github.com/landers1037
   Copyright Renj
*/

package serror

// 内部错误码对照

const (
	// SandwichInternalFlag 标识符
	SandwichInternalFlag = "X-Sandwich-Request-Flag"
	// SandwichBucketLimit 熔断
	SandwichBucketLimit = "SandwichBucketLimit"
	// SandwichReqLimit 限流
	SandwichReqLimit = "SandwichReqLimit"
	// SandwichDomainNotAllow 域名不支持
	SandwichDomainNotAllow = "SandwichDomainNotAllow"
	// SandwichBackendError 后端服务异常 针对API类服务异常
	SandwichBackendError = "SandwichBackendError"
)
