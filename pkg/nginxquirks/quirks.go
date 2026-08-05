// Package nginxquirks detects Ingress-NGINX behavioral semantics that naïve
// Gateway API conversion can break (see Kubernetes blog 2026-02-27).
package nginxquirks

import (
	"fmt"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/gateshift/gateshift/pkg/ir"
)

const (
	AnnUseRegex      = "nginx.ingress.kubernetes.io/use-regex"
	AnnRewriteTarget = "nginx.ingress.kubernetes.io/rewrite-target"
)

// HostMode describes Ingress-NGINX regex side effects for a hostname.
type HostMode struct {
	Host             string
	UseRegex         bool
	RewriteImplies   bool
	ForcingIngresses []string
}

// Result is the quirk analysis for a set of Ingresses.
type Result struct {
	HostModes map[string]HostMode
	Findings  []ir.AuditFinding
}

// Analyze inspects Ingresses for surprising Ingress-NGINX behaviors.
func Analyze(ingresses []*networkingv1.Ingress) Result {
	res := Result{HostModes: map[string]HostMode{}}
	if len(ingresses) == 0 {
		return res
	}

	for _, ing := range ingresses {
		if ing == nil {
			continue
		}
		useRegex := isTruthy(ing.Annotations[AnnUseRegex])
		rewrite := strings.TrimSpace(ing.Annotations[AnnRewriteTarget]) != ""
		if !useRegex && !rewrite {
			continue
		}
		for _, host := range hostsOf(ing) {
			m := res.HostModes[host]
			m.Host = host
			if useRegex {
				m.UseRegex = true
			}
			if rewrite {
				m.RewriteImplies = true
			}
			m.ForcingIngresses = appendUnique(m.ForcingIngresses, ing.Name)
			res.HostModes[host] = m
		}
	}

	seenHostFinding := map[string]bool{}
	trailingNoted := map[string]bool{}
	pathAsRegexNoted := map[string]bool{}

	for _, ing := range ingresses {
		if ing == nil {
			continue
		}
		ns := ing.Namespace
		if ns == "" {
			ns = "default"
		}

		for host, mode := range res.HostModes {
			if seenHostFinding[host] {
				continue
			}
			attr := mode.ForcingIngresses[0]
			if attr == "" {
				attr = ing.Name
			}
			seenHostFinding[host] = true
			res.Findings = append(res.Findings,
				ir.NewFinding(ir.FindingIDQuirkHostRegex, ir.StatusRequiresPolicy, 2,
					"gateshift.io/nginx-quirk/host-regex", buildHostRegexMessage(mode)).
					WithValue(host).
					WithTarget("HTTPRoute path RegularExpression fidelity").
					WithEvidence(ir.Evidence{IngressName: attr, Namespace: ns, Host: host}).
					WithFix("--preserve-nginx-regex"),
			)
		}

		for _, rule := range ing.Spec.Rules {
			host := rule.Host
			if host == "" {
				host = "*"
			}
			if rule.HTTP == nil {
				continue
			}
			forced := res.HostForcesRegex(host)
			for _, p := range rule.HTTP.Paths {
				path := p.Path
				if path == "" {
					path = "/"
				}
				pt := networkingv1.PathTypePrefix
				if p.PathType != nil {
					pt = *p.PathType
				}

				if forced && (pt == networkingv1.PathTypeExact || pt == networkingv1.PathTypePrefix) {
					pkey := ing.Name + "|" + host + "|" + string(pt) + "|" + path
					if !pathAsRegexNoted[pkey] {
						pathAsRegexNoted[pkey] = true
						res.Findings = append(res.Findings,
							ir.NewFinding(ir.FindingIDQuirkPathAsRegex, ir.StatusRequiresPolicy, 2,
								"gateshift.io/nginx-quirk/path-as-regex",
								fmt.Sprintf("On host %q, Ingress-NGINX treats this %s path as a case-insensitive prefix regex because regex mode is forced for the host. Naïve Exact/Prefix Gateway conversion may 404.", host, pt)).
								WithValue(fmt.Sprintf("%s %s", pt, path)).
								WithTarget("HTTPRoute.spec.rules[].matches[].path").
								WithEvidence(ir.Evidence{IngressName: ing.Name, Namespace: ns, Host: host, Path: path}).
								WithFix("--preserve-nginx-regex"),
						)
					}
				}

				if strings.HasSuffix(path, "/") && path != "/" &&
					(pt == networkingv1.PathTypeExact || pt == networkingv1.PathTypePrefix) {
					key := ing.Name + "|" + host + "|" + path
					if trailingNoted[key] {
						continue
					}
					trailingNoted[key] = true
					res.Findings = append(res.Findings,
						ir.NewFinding(ir.FindingIDQuirkTrailingSlash, ir.StatusRequiresPolicy, 2,
							"gateshift.io/nginx-quirk/trailing-slash",
							fmt.Sprintf("Ingress-NGINX redirects %s → %s with 301 when only a trailing slash differs. Gateway API does not.", strings.TrimSuffix(path, "/"), path)).
							WithValue(path).
							WithTarget("HTTPRoute RequestRedirect (optional)").
							WithEvidence(ir.Evidence{IngressName: ing.Name, Namespace: ns, Host: host, Path: path}).
							WithFix("--emit-trailing-slash-redirects"),
					)
				}
			}
		}
	}

	ing := ingresses[0]
	ns := ing.Namespace
	if ns == "" {
		ns = "default"
	}
	res.Findings = append(res.Findings,
		ir.NewFinding(ir.FindingIDQuirkURLNormalization, ir.StatusDirect, 1,
			"gateshift.io/nginx-quirk/url-normalization",
			"Ingress-NGINX normalizes '.', '..', and '//' before match. Envoy Gateway / Istio / Kgateway typically normalize '.'/'..' by default — verify slash-collapse and trailing-slash interaction for your controller.").
			WithTarget("Gateway implementation path normalization").
			WithEvidence(ir.Evidence{IngressName: ing.Name, Namespace: ns}),
	)

	return res
}

// HostForcesRegex reports whether Ingress-NGINX regex semantics apply to host.
func (r Result) HostForcesRegex(host string) bool {
	if host == "" {
		host = "*"
	}
	m, ok := r.HostModes[host]
	return ok && (m.UseRegex || m.RewriteImplies)
}

// PreserveRegexPath rewrites a path into a Gateway RegularExpression approximating
// Ingress-NGINX case-insensitive prefix matching.
func PreserveRegexPath(path string) string {
	if path == "" {
		path = "/"
	}
	if strings.HasPrefix(path, "(?i)") {
		return ensurePrefixWildcard(path)
	}
	if looksLikeRegex(path) {
		return ensurePrefixWildcard("(?i)" + path)
	}
	return "(?i)" + escapeLiteral(path) + ".*"
}

// TrailingSlashRedirect returns (withoutSlash, withSlash) when a redirect should be emitted.
func TrailingSlashRedirect(path string, pathType *networkingv1.PathType) (string, string, bool) {
	if path == "" || path == "/" || !strings.HasSuffix(path, "/") {
		return "", "", false
	}
	pt := networkingv1.PathTypePrefix
	if pathType != nil {
		pt = *pathType
	}
	if pt != networkingv1.PathTypeExact && pt != networkingv1.PathTypePrefix {
		return "", "", false
	}
	return strings.TrimSuffix(path, "/"), path, true
}

func ensurePrefixWildcard(pattern string) string {
	if strings.HasSuffix(pattern, ".*") || strings.HasSuffix(pattern, "$") {
		return pattern
	}
	return pattern + ".*"
}

func looksLikeRegex(path string) bool {
	for _, c := range path {
		switch c {
		case '*', '+', '?', '[', ']', '(', ')', '{', '}', '|', '\\', '^', '$':
			return true
		}
	}
	return false
}

func escapeLiteral(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '.', '+', '*', '?', '(', ')', '|', '[', ']', '{', '}', '^', '$', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func buildHostRegexMessage(m HostMode) string {
	why := []string{}
	if m.UseRegex {
		why = append(why, "use-regex=true")
	}
	if m.RewriteImplies {
		why = append(why, "rewrite-target (implies regex)")
	}
	return fmt.Sprintf(
		"Ingress-NGINX forces case-insensitive prefix regex for host %q (%s) across all Ingresses on that host (sources: %s). Envoy-based Gateway controllers usually do full case-sensitive regex. Convert with --preserve-nginx-regex or consciously accept semantic change.",
		m.Host, strings.Join(why, ", "), strings.Join(m.ForcingIngresses, ", "),
	)
}

func hostsOf(ing *networkingv1.Ingress) []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range ing.Spec.Rules {
		h := r.Host
		if h == "" {
			h = "*"
		}
		if !seen[h] {
			seen[h] = true
			out = append(out, h)
		}
	}
	if len(out) == 0 {
		out = []string{"*"}
	}
	return out
}

func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}

func appendUnique(ss []string, s string) []string {
	for _, x := range ss {
		if x == s {
			return ss
		}
	}
	return append(ss, s)
}
