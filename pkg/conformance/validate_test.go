package conformance

import (
	"testing"

	"github.com/gateshift/gateshift/pkg/ir"
)

func TestValidateBundleFlagsUntranslatable(t *testing.T) {
	bundle := &ir.MigrationBundle{
		HTTPRoutes: []ir.HTTPRouteIR{{
			Name: "x",
			Rules: []ir.HTTPRouteRuleIR{{
				Filters: []ir.FilterIR{{Kind: ir.FilterURLRewrite}},
			}},
		}},
		Findings: []ir.AuditFinding{{
			Key:     "nginx.ingress.kubernetes.io/configuration-snippet",
			Status:  ir.StatusUntranslatable,
			Level:   3,
			Message: "manual",
		}},
	}
	res := ValidateBundle(bundle, ir.ProviderEnvoyGateway)
	if res.OK {
		t.Fatal("expected failure when L3 findings present")
	}
}
