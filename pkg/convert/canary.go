package convert

import (
	"fmt"
	"strconv"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/gateshift/gateshift/pkg/ir"
	"github.com/gateshift/gateshift/pkg/nginxquirks"
)

const (
	annCanary            = "nginx.ingress.kubernetes.io/canary"
	annCanaryWeight      = "nginx.ingress.kubernetes.io/canary-weight"
	annCanaryHeader      = "nginx.ingress.kubernetes.io/canary-by-header"
	annCanaryHeaderValue = "nginx.ingress.kubernetes.io/canary-by-header-value"
)

// FromIngresses runs the ordered middle-layer pipeline (see pipeline.go):
// host-index/quirks → adapters/route build → canary merge → finalize IR.
func FromIngresses(ingresses []*networkingv1.Ingress, opts Options) (*ir.MigrationBundle, error) {
	if len(ingresses) == 0 {
		return nil, fmt.Errorf("no ingresses provided")
	}
	// Stage 1: host-index + behavioral quirks (before per-Ingress convert).
	quirks := nginxquirks.Analyze(ingresses)
	opts.quirkHostRegex = map[string]bool{}
	for host, mode := range quirks.HostModes {
		if mode.UseRegex || mode.RewriteImplies {
			opts.quirkHostRegex[host] = true
		}
	}

	// Stage 2: canary split.
	primary, canaries := splitCanaries(ingresses)
	var bundle *ir.MigrationBundle
	var err error
	if len(canaries) == 0 {
		// Stages 3–4: adapters + route build for each Ingress.
		bundle, err = mergeBundles(ingresses, opts)
	} else {
		baseOpts := opts
		if baseOpts.GatewayName == "" {
			baseOpts.GatewayName = primary.Name + "-gateway"
		}
		bundle, err = fromIngressRaw(primary, baseOpts)
		if err != nil {
			return nil, err
		}

		// Stage 5: canary merge.
		for _, c := range canaries {
			weight := 0
			if v := c.Annotations[annCanaryWeight]; v != "" {
				if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
					weight = n
				}
			}
			header := c.Annotations[annCanaryHeader]
			headerValue := c.Annotations[annCanaryHeaderValue]
			applyCanary(bundle, primary, c, weight, header, headerValue)
			ns := primary.Namespace
			if ns == "" {
				ns = "default"
			}
			bundle.Findings = append(bundle.Findings,
				ir.NewFinding(ir.FindingIDCanaryMerge, ir.StatusRequiresPolicy, 2, annCanary,
					fmt.Sprintf("Merged canary Ingress %s into primary %s", c.Name, primary.Name)).
					WithValue(fmt.Sprintf("weight=%d", weight)).
					WithTarget("HTTPRoute.spec.rules[].backendRefs.weight").
					WithEvidence(ir.Evidence{IngressName: primary.Name, Namespace: ns, Annotation: annCanary}),
			)
		}
	}
	if err != nil {
		return nil, err
	}
	// Stage 6: attach quirk findings (preserve/emit findings come from route build).
	bundle.Findings = append(bundle.Findings, quirks.Findings...)
	// Stage 7: normalize + RequiredFeatures.
	FinalizeIR(bundle)
	return bundle, nil
}

func mergeBundles(ingresses []*networkingv1.Ingress, opts Options) (*ir.MigrationBundle, error) {
	sharedGW := opts.GatewayName
	if sharedGW == "" && len(ingresses) > 1 {
		sharedGW = "shared-gateway"
		opts.GatewayName = sharedGW
	}
	combined := &ir.MigrationBundle{}
	includeGW := opts.IncludeGateway
	for i, ing := range ingresses {
		o := opts
		o.IncludeGateway = includeGW && i == 0
		if sharedGW != "" {
			o.GatewayName = sharedGW
		}
		b, err := fromIngressRaw(ing, o)
		if err != nil {
			return nil, err
		}
		combined.Findings = append(combined.Findings, b.Findings...)
		combined.HTTPRoutes = append(combined.HTTPRoutes, b.HTTPRoutes...)
		combined.Gateways = append(combined.Gateways, b.Gateways...)
		combined.Policies = append(combined.Policies, b.Policies...)
		combined.Certificates = append(combined.Certificates, b.Certificates...)
		if combined.SourceNamespace == "" {
			combined.SourceNamespace = b.SourceNamespace
		}
	}
	return combined, nil
}

func splitCanaries(ingresses []*networkingv1.Ingress) (primary *networkingv1.Ingress, canaries []*networkingv1.Ingress) {
	var rest []*networkingv1.Ingress
	for _, ing := range ingresses {
		if isTruthyAnn(ing.Annotations[annCanary]) {
			canaries = append(canaries, ing)
			continue
		}
		rest = append(rest, ing)
	}
	if len(rest) == 0 {
		// Degenerate: only canaries — treat first as primary.
		if len(canaries) == 0 {
			return nil, nil
		}
		return canaries[0], canaries[1:]
	}
	// Prefer primary that shares host with a canary.
	if len(canaries) > 0 {
		chosts := hostSet(canaries[0])
		for _, p := range rest {
			for h := range hostSet(p) {
				if chosts[h] {
					var others []*networkingv1.Ingress
					for _, x := range rest {
						if x != p {
							others = append(others, x)
						}
					}
					// Non-paired ingresses are ignored here; caller may convert separately.
					_ = others
					return p, canaries
				}
			}
		}
	}
	return rest[0], canaries
}

func applyCanary(bundle *ir.MigrationBundle, primary, canary *networkingv1.Ingress, weight int, header, headerValue string) {
	if len(bundle.HTTPRoutes) == 0 {
		return
	}
	route := &bundle.HTTPRoutes[0]
	wPrimary := int32(100 - weight)
	if wPrimary < 0 {
		wPrimary = 0
	}
	wCanary := int32(weight)

	canaryBackends := backendsFromIngress(canary)
	if len(canaryBackends) == 0 {
		return
	}

	if header != "" {
		// Header-based canary: add a higher-priority rule with header match.
		for _, be := range canaryBackends {
			match := ir.HTTPMatchIR{PathType: "PathPrefix", PathValue: be.path, Headers: map[string]string{}}
			if headerValue != "" {
				match.Headers[header] = headerValue
			} else {
				match.Headers[header] = "always"
			}
			wt := wCanary
			route.Rules = append([]ir.HTTPRouteRuleIR{{
				Matches:  []ir.HTTPMatchIR{match},
				Backends: []ir.BackendRefIR{{Name: be.name, Port: be.port, Weight: &wt}},
			}}, route.Rules...)
		}
		return
	}

	// Weight-based canary: add weighted backendRefs on matching paths.
	for i := range route.Rules {
		rule := &route.Rules[i]
		if len(rule.Backends) == 0 {
			continue
		}
		path := "/"
		if len(rule.Matches) > 0 {
			path = rule.Matches[0].PathValue
		}
		for _, cb := range canaryBackends {
			if !pathCompatible(path, cb.path) {
				continue
			}
			// Set primary weight if missing.
			if rule.Backends[0].Weight == nil {
				w := wPrimary
				rule.Backends[0].Weight = &w
			}
			wt := wCanary
			rule.Backends = append(rule.Backends, ir.BackendRefIR{Name: cb.name, Port: cb.port, Weight: &wt})
		}
	}
	_ = primary
}

type backendPath struct {
	name string
	port int32
	path string
}

func backendsFromIngress(ing *networkingv1.Ingress) []backendPath {
	var out []backendPath
	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, p := range rule.HTTP.Paths {
			if p.Backend.Service == nil {
				continue
			}
			path := p.Path
			if path == "" {
				path = "/"
			}
			port := p.Backend.Service.Port.Number
			if port == 0 {
				port = 80
			}
			out = append(out, backendPath{name: p.Backend.Service.Name, port: port, path: path})
		}
	}
	return out
}

func pathCompatible(a, b string) bool {
	return a == b || a == "/" || b == "/" || strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
}

func hostSet(ing *networkingv1.Ingress) map[string]bool {
	m := map[string]bool{}
	for _, r := range ing.Spec.Rules {
		if r.Host != "" {
			m[r.Host] = true
		}
	}
	return m
}

func isTruthyAnn(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}
