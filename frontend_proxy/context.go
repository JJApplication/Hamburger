package frontend_proxy

import "context"

type siteNameKey struct{}

// WithSiteName attaches the Helios site name to the request context.
// Gateway in-process static serving uses this instead of an internal header.
func WithSiteName(ctx context.Context, name string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, siteNameKey{}, name)
}

// SiteNameFrom returns the site name stored by WithSiteName.
func SiteNameFrom(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	name, ok := ctx.Value(siteNameKey{}).(string)
	if !ok || name == "" {
		return "", false
	}
	return name, true
}
