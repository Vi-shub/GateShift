package nginx

import (
	"fmt"
	"strings"

	"github.com/gateshift/gateshift/pkg/adapters"
	"github.com/gateshift/gateshift/pkg/ir"
	"github.com/gateshift/gateshift/pkg/patterns"
)

// SnippetAdapter uses the curated pattern library to promote safe idioms out of
// configuration-snippet / server-snippet. Residual nginx magic stays L3.
type SnippetAdapter struct{}

func (SnippetAdapter) Name() string          { return "snippets" }
func (SnippetAdapter) Level() adapters.Level { return adapters.Level3 }
func (SnippetAdapter) CanHandle(key string) bool {
	return key == AnnConfigurationSnippet || key == AnnServerSnippet || key == AnnModsecuritySnippet
}

func (SnippetAdapter) Transform(key, value string, ctx *adapters.Context) error {
	if key == AnnModsecuritySnippet {
		ctx.AddFinding(key, truncate(value, 120), ir.StatusUntranslatable, adapters.Level3, "",
			"ModSecurity snippets have no Gateway API equivalent — keep WAF at Gateway/controller layer")
		return nil
	}

	result := patterns.AnalyzeSnippet(value, ctx.Provider, ctx.Meta.IngressName, ctx.Meta.Namespace)

	// Always attach promoted filters/policies when patterns fire.
	ctx.Filters = append(ctx.Filters, result.Filters...)
	ctx.Policies = append(ctx.Policies, result.Policies...)

	switch {
	case result.FullyCovered:
		ctx.AddFinding(key, truncate(value, 120), ir.StatusDirect, adapters.Level1,
			"HTTPRoute filters via pattern library",
			fmt.Sprintf("Snippet fully matched by patterns (%d idioms, coverage=%.0f%%): %s",
				len(result.Matches), result.CoverageRatio*100, strings.Join(result.Hints, "; ")))
	case len(result.Matches) > 0:
		// Partial promotion — still needs human review for residual lines.
		msg := fmt.Sprintf("Partial snippet promotion (coverage=%.0f%%). Residual: %s",
			result.CoverageRatio*100, strings.Join(result.UnmatchedLines, " | "))
		if len(result.Hints) > 0 {
			msg += " | " + strings.Join(result.Hints, "; ")
		}
		ctx.AddFinding(key, truncate(value, 120), ir.StatusRequiresPolicy, adapters.Level2,
			"mixed pattern promotion + residual L3", msg)
	default:
		msg := "NGINX snippets/Lua cannot be auto-translated; rewrite as Gateway filters or controller policies manually"
		if len(result.Hints) > 0 {
			msg += " | hints: " + strings.Join(result.Hints, "; ")
		}
		ctx.AddFinding(key, truncate(value, 120), ir.StatusUntranslatable, adapters.Level3, "", msg)
	}
	return nil
}

// AuthAdapter flags external auth annotations as Level 3 (or Level 2 scaffold for Envoy).
type AuthAdapter struct{}

func (AuthAdapter) Name() string          { return "auth" }
func (AuthAdapter) Level() adapters.Level { return adapters.Level3 }
func (AuthAdapter) CanHandle(key string) bool {
	return key == AnnAuthURL || key == AnnAuthSignin || key == AnnAuthTLSSecret
}

func (AuthAdapter) Transform(key, value string, ctx *adapters.Context) error {
	if key == AnnAuthTLSSecret {
		ctx.AddFinding(key, value, ir.StatusRequiresPolicy, adapters.Level2,
			"BackendTLSPolicy / client cert auth",
			"mTLS client auth maps to BackendTLSPolicy or Gateway listener clientCertificateRef")
		return nil
	}
	if ctx.Provider == ir.ProviderEnvoyGateway && (key == AnnAuthURL || key == AnnAuthSignin) {
		// ExtAuth requires real backendRefs; do not emit an empty SecurityPolicy that fails apply.
		ctx.AddFinding(key, value, ir.StatusRequiresPolicy, adapters.Level2,
			"SecurityPolicy.extAuth",
			"auth-url/signin needs a SecurityPolicy.extAuth with backendRefs to your auth service (manual)")
		ctx.Claim(AnnAuthURL, AnnAuthSignin)
		return nil
	}
	ctx.AddFinding(key, value, ir.StatusUntranslatable, adapters.Level3, "",
		"External auth has no portable Gateway API Core equivalent; use controller SecurityPolicy/WASM")
	return nil
}
