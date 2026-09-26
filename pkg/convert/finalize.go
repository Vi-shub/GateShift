package convert

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/gateshift/gateshift/pkg/ir"
)

// FinalizeIR normalizes findings, sorts for determinism, and annotates features.
// Call at the end of every conversion path so emitters see a stable IR.
func FinalizeIR(bundle *ir.MigrationBundle) {
	if bundle == nil {
		return
	}
	ir.NormalizeFindings(bundle.Findings)
	sort.SliceStable(bundle.Findings, func(i, j int) bool {
		a, b := bundle.Findings[i], bundle.Findings[j]
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		if a.Key != b.Key {
			return a.Key < b.Key
		}
		if a.IngressName != b.IngressName {
			return a.IngressName < b.IngressName
		}
		return a.Value < b.Value
	})
	ir.AnnotateRequiredFeatures(bundle)
}

// StripExtensionPolicies removes BackendTrafficPolicy and SecurityPolicy from bundle.
// Use when Envoy Gateway CRDs are not installed on the cluster.
func StripExtensionPolicies(bundle *ir.MigrationBundle) {
	if bundle == nil {
		return
	}
	filtered := make([]ir.PolicyIR, 0, len(bundle.Policies))
	for _, pol := range bundle.Policies {
		kind, _ := pol.Spec["kind"].(string)
		apiVersion, _ := pol.Spec["apiVersion"].(string)
		// Skip Envoy Gateway extension policies
		if strings.HasPrefix(apiVersion, "gateway.envoyproxy.io/") {
			switch kind {
			case "BackendTrafficPolicy", "SecurityPolicy", "ClientTrafficPolicy":
				continue
			}
		}
		filtered = append(filtered, pol)
	}
	bundle.Policies = filtered
}

// MarshalIRJSON returns canonical IR JSON for golden tests.
func MarshalIRJSON(bundle *ir.MigrationBundle) ([]byte, error) {
	FinalizeIR(bundle)
	return json.MarshalIndent(bundle, "", "  ")
}
