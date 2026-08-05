// Package conformance checks whether a MigrationBundle uses features supported
// by the target Gateway controller (Core vs Extended capability matrix).
package conformance

import (
	"fmt"
	"strings"

	"github.com/gateshift/gateshift/pkg/ir"
)

// Feature is a Gateway API capability GateShift may emit.
type Feature string

const (
	FeatureHTTPRoute              Feature = "HTTPRoute"
	FeatureURLRewrite             Feature = "HTTPRouteURLRewrite"
	FeatureRequestRedirect        Feature = "HTTPRouteRequestRedirect"
	FeatureResponseHeaderModifier Feature = "HTTPRouteResponseHeaderModifier"
	FeatureRegularExpressionPath  Feature = "HTTPRouteMethodMatching" // placeholder grouping
	FeatureRegexPath              Feature = "HTTPRoutePathRegex"
	FeatureSessionPersistence     Feature = "HTTPRouteSessionPersistence"
	FeatureBackendTLS             Feature = "BackendTLSPolicy"
)

// Profile describes what a provider/controller supports.
type Profile struct {
	Provider     ir.Provider
	GatewayClass string
	Supported    map[Feature]bool
	Notes        map[Feature]string
}

// DefaultProfiles returns built-in capability maps for common controllers.
func DefaultProfiles() map[ir.Provider]Profile {
	core := map[Feature]bool{
		FeatureHTTPRoute:              true,
		FeatureRequestRedirect:        true,
		FeatureResponseHeaderModifier: true,
	}
	return map[ir.Provider]Profile{
		ir.ProviderStandard: {
			Provider:  ir.ProviderStandard,
			Supported: merge(core, map[Feature]bool{FeatureURLRewrite: true}),
		},
		ir.ProviderEnvoyGateway: {
			Provider:     ir.ProviderEnvoyGateway,
			GatewayClass: "envoy",
			Supported: merge(core, map[Feature]bool{
				FeatureURLRewrite:         true,
				FeatureRegexPath:          true,
				FeatureSessionPersistence: true,
			}),
		},
		ir.ProviderCilium: {
			Provider:     ir.ProviderCilium,
			GatewayClass: "cilium",
			Supported: merge(core, map[Feature]bool{
				FeatureURLRewrite: true,
				FeatureRegexPath:  true,
			}),
			Notes: map[Feature]string{
				FeatureSessionPersistence: "Limited / check Cilium version",
			},
		},
		ir.ProviderIstio: {
			Provider:     ir.ProviderIstio,
			GatewayClass: "istio",
			Supported: merge(core, map[Feature]bool{
				FeatureURLRewrite: true,
				FeatureRegexPath:  true,
			}),
		},
		ir.ProviderKong: {
			Provider:     ir.ProviderKong,
			GatewayClass: "kong",
			Supported: merge(core, map[Feature]bool{
				FeatureURLRewrite: true,
			}),
			Notes: map[Feature]string{
				FeatureRegexPath: "Prefer Kong plugins for complex regex",
			},
		},
	}
}

func merge(a, b map[Feature]bool) map[Feature]bool {
	out := map[Feature]bool{}
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}

// Issue is a conformance problem found in a bundle.
type Issue struct {
	Feature  Feature
	Severity string // warn | error
	Message  string
}

// Result aggregates conformance findings.
type Result struct {
	Provider ir.Provider
	Profile  Profile
	Issues   []Issue
	OK       bool
}

// ValidateBundle checks IR features against a provider profile.
func ValidateBundle(bundle *ir.MigrationBundle, provider ir.Provider) Result {
	profiles := DefaultProfiles()
	profile, ok := profiles[provider]
	if !ok {
		profile = profiles[ir.ProviderStandard]
	}
	res := Result{Provider: provider, Profile: profile, OK: true}

	used := detectFeatures(bundle)
	for feat := range used {
		if profile.Supported[feat] {
			continue
		}
		sev := "error"
		msg := fmt.Sprintf("Feature %s is not in the %s support matrix", feat, provider)
		if note, has := profile.Notes[feat]; has {
			sev = "warn"
			msg = msg + " (" + note + ")"
		}
		res.Issues = append(res.Issues, Issue{Feature: feat, Severity: sev, Message: msg})
		if sev == "error" {
			res.OK = false
		}
	}

	// Untranslatable findings always warn.
	for _, f := range bundle.Findings {
		if f.Status == ir.StatusUntranslatable {
			res.Issues = append(res.Issues, Issue{
				Feature:  Feature("Annotation:" + f.Key),
				Severity: "error",
				Message:  f.Message,
			})
			res.OK = false
		}
		if f.Status == ir.StatusRequiresPolicy {
			res.Issues = append(res.Issues, Issue{
				Feature:  Feature("Policy:" + f.Key),
				Severity: "warn",
				Message:  "Requires provider Policy CRD — ensure " + string(provider) + " CRDs are installed",
			})
		}
	}
	return res
}

func detectFeatures(bundle *ir.MigrationBundle) map[Feature]bool {
	used := map[Feature]bool{FeatureHTTPRoute: true}
	for _, route := range bundle.HTTPRoutes {
		for _, rule := range route.Rules {
			for _, m := range rule.Matches {
				if strings.EqualFold(m.PathType, "RegularExpression") {
					used[FeatureRegexPath] = true
				}
			}
			for _, f := range rule.Filters {
				switch f.Kind {
				case ir.FilterURLRewrite:
					used[FeatureURLRewrite] = true
				case ir.FilterRequestRedirect:
					used[FeatureRequestRedirect] = true
				case ir.FilterResponseHeader, ir.FilterRequestHeader:
					used[FeatureResponseHeaderModifier] = true
				}
			}
		}
	}
	for _, pol := range bundle.Policies {
		if pol.Kind == ir.PolicySessionAffinity {
			used[FeatureSessionPersistence] = true
		}
	}
	return used
}

// Format returns a human-readable conformance report.
func Format(r Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "GateShift Conformance Report\n")
	fmt.Fprintf(&b, "============================\n")
	fmt.Fprintf(&b, "Provider: %s\n", r.Provider)
	if r.Profile.GatewayClass != "" {
		fmt.Fprintf(&b, "Suggested GatewayClass: %s\n", r.Profile.GatewayClass)
	}
	if r.OK && len(r.Issues) == 0 {
		fmt.Fprintf(&b, "Result: PASS — all emitted features are supported\n")
		return b.String()
	}
	if r.OK {
		fmt.Fprintf(&b, "Result: PASS WITH WARNINGS\n\n")
	} else {
		fmt.Fprintf(&b, "Result: FAIL — do not apply until issues are resolved\n\n")
	}
	for _, issue := range r.Issues {
		fmt.Fprintf(&b, "[%s] %s — %s\n", strings.ToUpper(issue.Severity), issue.Feature, issue.Message)
	}
	return b.String()
}
