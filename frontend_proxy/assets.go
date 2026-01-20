package frontend_proxy

import _ "embed"

var (
	//go:embed assets/404.html
	PageNotFound []byte
	//go:embed assets/500.html
	PageInternalError []byte
	//go:embed assets/index.html
	PageIndex []byte
)
