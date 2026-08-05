package ir

import "strings"

// SchemaVersion is the IR contract version emitters/tests pin to.
const SchemaVersion = "gateshift.ir/v1"

// Feature is a capability the IR may require from a Gateway controller.
type Feature string

const (
	FeatureHTTPRoute          Feature = "HTTPRoute"
	FeatureURLRewrite         Feature = "HTTPRouteURLRewrite"
	FeatureRequestRedirect    Feature = "HTTPRouteRequestRedirect"
	FeatureHeaderModifier     Feature = "HTTPRouteHeaderModifier"
	FeatureRegexPath          Feature = "HTTPRoutePathRegex"
	FeatureSessionPersistence Feature = "HTTPRouteSessionPersistence"
	FeatureBackendTLS         Feature = "BackendTLSPolicy"
	FeatureProviderPolicy     Feature = "ProviderPolicy"
)

// CollectRequiredFeatures derives controller features from IR nodes.
func CollectRequiredFeatures(b *MigrationBundle) []Feature {
	if b == nil {
		return nil
	}
	used := map[Feature]bool{FeatureHTTPRoute: true}
	for _, route := range b.HTTPRoutes {
		for _, rule := range route.Rules {
			for _, m := range rule.Matches {
				if strings.EqualFold(m.PathType, "RegularExpression") {
					used[FeatureRegexPath] = true
				}
			}
			for _, f := range rule.Filters {
				switch f.Kind {
				case FilterURLRewrite:
					used[FeatureURLRewrite] = true
				case FilterRequestRedirect:
					used[FeatureRequestRedirect] = true
				case FilterResponseHeader, FilterRequestHeader:
					used[FeatureHeaderModifier] = true
				}
			}
		}
	}
	for _, pol := range b.Policies {
		used[FeatureProviderPolicy] = true
		if pol.Kind == PolicySessionAffinity {
			used[FeatureSessionPersistence] = true
		}
		if pol.Kind == PolicyBackendTuning {
			// backend TLS often rides BackendTuning / dedicated policies
			if _, ok := pol.Spec["tls"]; ok {
				used[FeatureBackendTLS] = true
			}
		}
	}
	out := make([]Feature, 0, len(used))
	order := []Feature{
		FeatureHTTPRoute, FeatureURLRewrite, FeatureRequestRedirect, FeatureHeaderModifier,
		FeatureRegexPath, FeatureSessionPersistence, FeatureBackendTLS, FeatureProviderPolicy,
	}
	for _, f := range order {
		if used[f] {
			out = append(out, f)
		}
	}
	return out
}

// AnnotateRequiredFeatures sets bundle.RequiredFeatures from IR contents.
func AnnotateRequiredFeatures(b *MigrationBundle) {
	if b == nil {
		return
	}
	b.SchemaVersion = SchemaVersion
	b.RequiredFeatures = CollectRequiredFeatures(b)
}
