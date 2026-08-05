package ir_test

import (
	"testing"

	"github.com/gateshift/gateshift/pkg/ir"
)

func TestNormalizeFindingsAndFeatures(t *testing.T) {
	b := &ir.MigrationBundle{
		HTTPRoutes: []ir.HTTPRouteIR{{
			Name: "r",
			Rules: []ir.HTTPRouteRuleIR{{
				Matches: []ir.HTTPMatchIR{{PathType: "RegularExpression", PathValue: "(?i)/x.*"}},
				Filters: []ir.FilterIR{{Kind: ir.FilterURLRewrite}},
			}},
		}},
		Findings: []ir.AuditFinding{{
			Key:     "nginx.ingress.kubernetes.io/foo",
			Status:  ir.StatusRequiresPolicy,
			Message: "x",
		}},
	}
	ir.NormalizeFindings(b.Findings)
	ir.AnnotateRequiredFeatures(b)
	if b.SchemaVersion != ir.SchemaVersion {
		t.Fatalf("schema %q", b.SchemaVersion)
	}
	if b.Findings[0].ID == "" || b.Findings[0].Severity != ir.SeverityWarn {
		t.Fatalf("normalize failed: %#v", b.Findings[0])
	}
	var hasRegex, hasRewrite bool
	for _, f := range b.RequiredFeatures {
		if f == ir.FeatureRegexPath {
			hasRegex = true
		}
		if f == ir.FeatureURLRewrite {
			hasRewrite = true
		}
	}
	if !hasRegex || !hasRewrite {
		t.Fatalf("features: %v", b.RequiredFeatures)
	}
}
