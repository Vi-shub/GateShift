package convert

import (
	"encoding/json"
	"sort"

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

// MarshalIRJSON returns canonical IR JSON for golden tests.
func MarshalIRJSON(bundle *ir.MigrationBundle) ([]byte, error) {
	FinalizeIR(bundle)
	return json.MarshalIndent(bundle, "", "  ")
}
