package nginx

import (
	"strings"

	"github.com/gateshift/gateshift/pkg/adapters"
	"github.com/gateshift/gateshift/pkg/ir"
)

// WWWRedirectAdapter maps from-to-www-redirect → RequestRedirect hostname.
type WWWRedirectAdapter struct{}

func (WWWRedirectAdapter) Name() string          { return "www-redirect" }
func (WWWRedirectAdapter) Level() adapters.Level { return adapters.Level1 }
func (WWWRedirectAdapter) CanHandle(key string) bool {
	return key == AnnFromToWWWRedirect
}

func (WWWRedirectAdapter) Transform(key, value string, ctx *adapters.Context) error {
	if !isTruthy(value) {
		ctx.AddFinding(key, value, ir.StatusDirect, adapters.Level1, "", "www redirect disabled")
		return nil
	}
	status := 308
	// Hostname is filled by convert using Ingress hosts; emit placeholder filter.
	host := "www."
	ctx.Filters = append(ctx.Filters, ir.FilterIR{
		Kind:               ir.FilterRequestRedirect,
		RedirectHostname:   &host,
		RedirectStatusCode: &status,
	})
	ctx.AddFinding(key, value, ir.StatusDirect, adapters.Level1,
		"HTTPRoute.spec.rules[].filters[type=RequestRedirect]",
		"from-to-www-redirect mapped to RequestRedirect (convert fills concrete hostname)")
	return nil
}

// AppRootAdapter maps app-root → RequestRedirect to a path.
type AppRootAdapter struct{}

func (AppRootAdapter) Name() string          { return "app-root" }
func (AppRootAdapter) Level() adapters.Level { return adapters.Level1 }
func (AppRootAdapter) CanHandle(key string) bool {
	return key == AnnAppRoot
}

func (AppRootAdapter) Transform(key, value string, ctx *adapters.Context) error {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "/"
	}
	status := 302
	ctx.Filters = append(ctx.Filters, ir.FilterIR{
		Kind:               ir.FilterRequestRedirect,
		RedirectPath:       &value,
		RedirectPathType:   "ReplaceFullPath",
		RedirectStatusCode: &status,
	})
	ctx.AddFinding(key, value, ir.StatusDirect, adapters.Level1,
		"HTTPRoute.spec.rules[].filters[type=RequestRedirect]",
		"app-root mapped to RequestRedirect path")
	return nil
}

// XForwardedPrefixAdapter maps x-forwarded-prefix → RequestHeaderModifier.
type XForwardedPrefixAdapter struct{}

func (XForwardedPrefixAdapter) Name() string          { return "x-forwarded-prefix" }
func (XForwardedPrefixAdapter) Level() adapters.Level { return adapters.Level1 }
func (XForwardedPrefixAdapter) CanHandle(key string) bool {
	return key == AnnXForwardedPrefix
}

func (XForwardedPrefixAdapter) Transform(key, value string, ctx *adapters.Context) error {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "true"
	}
	// When value is "true", nginx sets X-Forwarded-Prefix from the location path;
	// we set a literal header the operator can adjust.
	headerVal := value
	if isTruthy(value) {
		headerVal = "/"
	}
	ctx.Filters = append(ctx.Filters, ir.FilterIR{
		Kind: ir.FilterRequestHeader,
		SetHeaders: map[string]string{
			"X-Forwarded-Prefix": headerVal,
		},
	})
	ctx.AddFinding(key, value, ir.StatusDirect, adapters.Level1,
		"HTTPRoute.spec.rules[].filters[type=RequestHeaderModifier]",
		"x-forwarded-prefix mapped to RequestHeaderModifier")
	return nil
}
