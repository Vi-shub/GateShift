// Package diff produces structural comparisons between Ingress and HTTPRoute IR.
package diff

import (
	"fmt"
	"io"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/gateshift/gateshift/pkg/ir"
)

// WriteSideBySide prints a simple structural diff of Ingress vs translated HTTPRoute.
func WriteSideBySide(w io.Writer, ing *networkingv1.Ingress, bundle *ir.MigrationBundle) {
	fmt.Fprintln(w, "GateShift Structural Diff")
	fmt.Fprintln(w, "=========================")
	fmt.Fprintf(w, "Source Ingress: %s/%s\n\n", ing.Namespace, ing.Name)

	left := ingressSummary(ing)
	right := routeSummary(bundle)

	max := len(left)
	if len(right) > max {
		max = len(right)
	}
	fmt.Fprintf(w, "%-48s | %s\n", "INGRESS", "GATEWAY API")
	fmt.Fprintf(w, "%s-+-%s\n", strings.Repeat("-", 48), strings.Repeat("-", 48))
	for i := 0; i < max; i++ {
		l, r := "", ""
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		marker := " "
		if l != r && l != "" && r != "" {
			marker = "*"
		}
		fmt.Fprintf(w, "%-48s %s %s\n", truncate(l, 48), marker, truncate(r, 48))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "* marks rows that differ in representation")
}

func ingressSummary(ing *networkingv1.Ingress) []string {
	var lines []string
	lines = append(lines, "kind: Ingress")
	lines = append(lines, "name: "+ing.Name)
	if ing.Spec.IngressClassName != nil && *ing.Spec.IngressClassName != "" {
		lines = append(lines, "ingressClass: "+*ing.Spec.IngressClassName)
	}
	for _, rule := range ing.Spec.Rules {
		host := rule.Host
		if host == "" {
			host = "*"
		}
		lines = append(lines, "host: "+host)
		if rule.HTTP == nil {
			continue
		}
		for _, p := range rule.HTTP.Paths {
			svc := "?"
			port := int32(0)
			if p.Backend.Service != nil {
				svc = p.Backend.Service.Name
				port = p.Backend.Service.Port.Number
			}
			pt := "Prefix"
			if p.PathType != nil {
				pt = string(*p.PathType)
			}
			lines = append(lines, fmt.Sprintf("  path[%s]: %s -> %s:%d", pt, p.Path, svc, port))
		}
	}
	for k, v := range ing.Annotations {
		if strings.HasPrefix(k, "nginx.ingress.kubernetes.io/") || strings.HasPrefix(k, "cert-manager.io/") {
			lines = append(lines, fmt.Sprintf("ann: %s=%s", k, truncate(v, 24)))
		}
	}
	return lines
}

func routeSummary(bundle *ir.MigrationBundle) []string {
	var lines []string
	for _, gw := range bundle.Gateways {
		lines = append(lines, "kind: Gateway")
		lines = append(lines, "name: "+gw.Name)
		lines = append(lines, "gatewayClass: "+gw.ClassName)
		for _, l := range gw.Listeners {
			lines = append(lines, fmt.Sprintf("listener: %s %s:%d", l.Name, l.Protocol, l.Port))
		}
	}
	for _, route := range bundle.HTTPRoutes {
		lines = append(lines, "kind: HTTPRoute")
		lines = append(lines, "name: "+route.Name)
		for _, h := range route.Hostnames {
			lines = append(lines, "hostname: "+h)
		}
		for _, rule := range route.Rules {
			for _, m := range rule.Matches {
				be := "?"
				port := int32(0)
				if len(rule.Backends) > 0 {
					be = rule.Backends[0].Name
					port = rule.Backends[0].Port
				}
				lines = append(lines, fmt.Sprintf("  match[%s]: %s -> %s:%d", m.PathType, m.PathValue, be, port))
			}
			for _, f := range rule.Filters {
				lines = append(lines, "  filter: "+string(f.Kind))
			}
		}
	}
	for _, pol := range bundle.Policies {
		lines = append(lines, fmt.Sprintf("policy: %s (%s)", pol.Name, pol.Kind))
	}
	return lines
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
