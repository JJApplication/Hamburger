package static

import _ "embed"

var (
	//go:embed forbidden.html
	ForbiddenPage []byte
	//go:embed unavailable.html
	UnavailablePage []byte
)
