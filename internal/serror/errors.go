/*
Project: Sandwich errors.go
Created: 2021/12/12 by Landers
*/

package serror

import "fmt"

const (
	ERRORSendProxy = "[Sandwich] proxy resolve error"
	ERRORTooMany   = "[Sandwich] too many request"
)

var (
	ErrorProxyErrorJSON = fmt.Sprintf("{\"error\":\"%s\"}", ERRORSendProxy)
)
