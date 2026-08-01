package nginx

import (
	"strings"

	"github.com/gateshift/gateshift/pkg/adapters"
	"github.com/gateshift/gateshift/pkg/ir"
)

// SSLRedirectAdapter maps ssl-redirect / force-ssl-redirect → RequestRedirect.
type SSLRedirectAdapter struct{}

func (SSLRedirectAdapter) Name() string          { return "ssl-redirect" }
func (SSLRedirectAdapter) Level() adapters.Level { return adapters.Level1 }
func (SSLRedirectAdapter) CanHandle(key string) bool {
	return key == AnnSSLRedirect || key == AnnForceSSLRedirect
}

func (SSLRedirectAdapter) Transform(key, value string, ctx *adapters.Context) error {
	if !isTruthy(value) {
		ctx.AddFinding(key, value, ir.StatusDirect, adapters.Level1, "", "ssl-redirect disabled; no filter emitted")
		return nil
	}
	// Deduplicate if both annotations present.
	for _, f := range ctx.Filters {
		if f.Kind == ir.FilterRequestRedirect && f.RedirectScheme != nil && *f.RedirectScheme == "https" {
			ctx.AddFinding(key, value, ir.StatusDirect, adapters.Level1,
				"HTTPRoute.spec.rules[].filters[type=RequestRedirect]",
				"HTTPS redirect already emitted")
			return nil
		}
	}
	status := 302
	scheme := "https"
	ctx.Filters = append(ctx.Filters, ir.FilterIR{
		Kind:               ir.FilterRequestRedirect,
		RedirectScheme:     &scheme,
		RedirectStatusCode: &status,
	})
	ctx.AddFinding(key, value, ir.StatusDirect, adapters.Level1,
		"HTTPRoute.spec.rules[].filters[type=RequestRedirect]",
		"Mapped SSL redirect to HTTPS RequestRedirect")
	return nil
}

// ExternalRedirectAdapter maps permanent/temporal-redirect → RequestRedirect.
type ExternalRedirectAdapter struct{}

func (ExternalRedirectAdapter) Name() string          { return "external-redirect" }
func (ExternalRedirectAdapter) Level() adapters.Level { return adapters.Level1 }
func (ExternalRedirectAdapter) CanHandle(key string) bool {
	return key == AnnPermanentRedirect || key == AnnTemporalRedirect
}

func (ExternalRedirectAdapter) Transform(key, value string, ctx *adapters.Context) error {
	value = strings.TrimSpace(value)
	if value == "" {
		ctx.AddFinding(key, value, ir.StatusUntranslatable, adapters.Level3, "", "Empty redirect URL")
		return nil
	}
	status := 302
	if key == AnnPermanentRedirect {
		status = 301
	}
	// Prefer full URL hostname/path split when possible; otherwise stash as hostname.
	hostname := value
	scheme := ""
	if strings.HasPrefix(value, "https://") {
		scheme = "https"
		hostname = strings.TrimPrefix(value, "https://")
	} else if strings.HasPrefix(value, "http://") {
		scheme = "http"
		hostname = strings.TrimPrefix(value, "http://")
	}
	path := "/"
	if i := strings.Index(hostname, "/"); i >= 0 {
		path = hostname[i:]
		hostname = hostname[:i]
	}
	f := ir.FilterIR{
		Kind:               ir.FilterRequestRedirect,
		RedirectStatusCode: &status,
		RedirectHostname:   &hostname,
		RedirectPath:       &path,
		RedirectPathType:   "ReplaceFullPath",
	}
	if scheme != "" {
		f.RedirectScheme = &scheme
	}
	ctx.Filters = append(ctx.Filters, f)
	ctx.AddFinding(key, value, ir.StatusDirect, adapters.Level1,
		"HTTPRoute.spec.rules[].filters[type=RequestRedirect]",
		"Mapped external redirect annotation to RequestRedirect")
	return nil
}
