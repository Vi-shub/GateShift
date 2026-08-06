package convert

import (
	"fmt"
	"strings"

	"github.com/gateshift/gateshift/pkg/ir"
)

const (
	AnnMode          = "gateshift.io/mode"
	AnnShadow        = "gateshift.io/shadow"
	AnnSourceIngress = "gateshift.io/source-ingress"
	ModeDualRun      = "dual-run"
)

// DualRunOptions controls staging naming for shadowed cutover.
type DualRunOptions struct {
	// GatewayName overrides the staging Gateway name (and parentRefs).
	// Empty → rename generated gateways with a -staging suffix.
	GatewayName string
	// IncludeGateway false keeps HTTPRoute/policies only (attach to existing Gateway).
	IncludeGateway bool
}

// ApplyDualRunMode rewrites a converted bundle for dual-run:
// Ingress stays live; emitted resources are staging/shadow and never delete Ingress.
func ApplyDualRunMode(bundle *ir.MigrationBundle, opts DualRunOptions) {
	if bundle == nil {
		return
	}

	stagingName := opts.GatewayName
	if opts.IncludeGateway {
		if stagingName == "" {
			if len(bundle.Gateways) > 0 {
				stagingName = stagingGatewayName(bundle.Gateways[0].Name)
			} else {
				stagingName = "staging-gateway"
			}
		}
		if len(bundle.Gateways) == 0 {
			// Convert ran with IncludeGateway false; synthesize a minimal staging parent
			// only when the caller asked for a gateway — dual-run CLI always converts with gateway on.
		} else {
			for i := range bundle.Gateways {
				gw := &bundle.Gateways[i]
				if i == 0 || opts.GatewayName != "" {
					gw.Name = stagingName
				} else {
					gw.Name = stagingGatewayName(gw.Name)
				}
				if gw.Annotations == nil {
					gw.Annotations = map[string]string{}
				}
				gw.Annotations[AnnMode] = ModeDualRun
				gw.Annotations[AnnShadow] = "true"
			}
			// Collapse to a single staging Gateway when multiple were emitted.
			if len(bundle.Gateways) > 1 && opts.GatewayName != "" {
				bundle.Gateways = bundle.Gateways[:1]
			}
		}
	} else {
		// Attach-only: drop generated Gateways; parentRef uses override or existing name.
		bundle.Gateways = nil
		if stagingName == "" {
			stagingName = "staging-gateway"
		}
	}

	for i := range bundle.HTTPRoutes {
		route := &bundle.HTTPRoutes[i]
		src := route.Name
		if route.Annotations != nil {
			if v := route.Annotations[AnnSourceIngress]; v != "" {
				src = v
			}
		}
		route.Name = shadowRouteName(src)
		if route.Annotations == nil {
			route.Annotations = map[string]string{}
		}
		route.Annotations[AnnMode] = ModeDualRun
		route.Annotations[AnnShadow] = "true"
		route.Annotations[AnnSourceIngress] = src

		parent := stagingName
		if parent == "" && len(route.ParentRefs) > 0 {
			parent = stagingGatewayName(route.ParentRefs[0].Name)
		}
		if len(route.ParentRefs) == 0 {
			route.ParentRefs = []ir.ParentRefIR{{Name: parent, Namespace: route.Namespace}}
		} else {
			for j := range route.ParentRefs {
				route.ParentRefs[j].Name = parent
			}
		}
	}

	// Policies that target the old route/gateway keep working via TargetRef name updates when present.
	for i := range bundle.Policies {
		pol := &bundle.Policies[i]
		if strings.HasSuffix(pol.TargetRef.Name, "-gateway") || pol.TargetRef.Name == stagingName {
			if stagingName != "" {
				pol.TargetRef.Name = stagingName
			}
		} else if pol.TargetRef.Name != "" {
			// Likely an HTTPRoute target — point at shadow route name.
			pol.TargetRef.Name = shadowRouteName(pol.TargetRef.Name)
		}
	}

	bundle.Findings = append(bundle.Findings,
		ir.NewFinding(ir.FindingIDDualRun, ir.StatusDirect, 1, "gateshift.io/mode",
			"Dual-run mode: Ingress is left unchanged; apply only the emitted staging Gateway / shadow HTTPRoute for parallel validation").
			WithValue(ModeDualRun).
			WithTarget("HTTPRoute + staging Gateway").
			WithFix("gateshift dual-run"),
	)
	FinalizeIR(bundle)
}

// FormatDualRunChecklist returns a human cutover checklist (print to stderr).
func FormatDualRunChecklist(bundle *ir.MigrationBundle) string {
	var b strings.Builder
	fmt.Fprintf(&b, "GateShift dual-run checklist\n")
	fmt.Fprintf(&b, "===========================\n")
	fmt.Fprintf(&b, "Ingress is NOT modified or deleted. Apply only the YAML below for shadowed traffic.\n\n")
	if bundle != nil {
		for _, gw := range bundle.Gateways {
			fmt.Fprintf(&b, "1. Apply staging Gateway %s/%s (GatewayClass=%s)\n", gw.Namespace, gw.Name, gw.ClassName)
		}
		if len(bundle.Gateways) == 0 {
			fmt.Fprintf(&b, "1. Ensure the parent Gateway named in parentRefs already exists\n")
		}
		for _, r := range bundle.HTTPRoutes {
			parent := ""
			if len(r.ParentRefs) > 0 {
				parent = r.ParentRefs[0].Name
			}
			src := r.Annotations[AnnSourceIngress]
			fmt.Fprintf(&b, "2. Apply shadow HTTPRoute %s/%s (source Ingress %q → parent Gateway %q)\n",
				r.Namespace, r.Name, src, parent)
		}
	}
	fmt.Fprintf(&b, "3. Send canary/shadow traffic via the staging Gateway address (not production DNS yet)\n")
	fmt.Fprintf(&b, "4. Compare behavior with the live Ingress (paths, redirects, TLS, backends)\n")
	fmt.Fprintf(&b, "5. When confident: flip DNS / listeners to the Gateway; delete Ingress last\n")
	fmt.Fprintf(&b, "6. Re-run without dual-run (`gateshift convert` / `migrate`) for the final GitOps manifests if needed\n")
	return b.String()
}

func stagingGatewayName(name string) string {
	if name == "" {
		return "staging-gateway"
	}
	if strings.HasSuffix(name, "-staging-gateway") {
		return name
	}
	if strings.HasSuffix(name, "-gateway") {
		return strings.TrimSuffix(name, "-gateway") + "-staging-gateway"
	}
	if strings.Contains(name, "staging") {
		return name
	}
	return name + "-staging"
}

func shadowRouteName(name string) string {
	if name == "" {
		return "shadow"
	}
	if strings.HasSuffix(name, "-shadow") {
		return name
	}
	return name + "-shadow"
}
