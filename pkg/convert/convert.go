// Package convert turns Kubernetes Ingress objects into a MigrationBundle IR
// and then into Gateway API-oriented unstructured YAML documents.
package convert

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/yaml"

	"github.com/gateshift/gateshift/pkg/adapters/nginx"
	"github.com/gateshift/gateshift/pkg/ir"
	"github.com/gateshift/gateshift/pkg/nginxquirks"
)

// Options controls conversion behavior.
type Options struct {
	Provider       ir.Provider
	GatewayName    string
	GatewayClass   string
	IncludeGateway bool

	// PreserveNGINXRegex rewrites paths on regex-forced hosts to case-insensitive
	// prefix RegularExpression matches (Ingress-NGINX semantics).
	PreserveNGINXRegex bool
	// EmitTrailingSlashRedirects adds 301 redirects from /path → /path/ where
	// Ingress-NGINX would auto-redirect.
	EmitTrailingSlashRedirects bool

	// quirkHostRegex is set by FromIngresses after multi-Ingress analysis.
	quirkHostRegex map[string]bool
}

// FromIngress converts a single Ingress into finalized IR.
func FromIngress(ing *networkingv1.Ingress, opts Options) (*ir.MigrationBundle, error) {
	bundle, err := fromIngressRaw(ing, opts)
	if err != nil {
		return nil, err
	}
	FinalizeIR(bundle)
	return bundle, nil
}

// fromIngressRaw builds IR without FinalizeIR (used by the multi-Ingress pipeline).
func fromIngressRaw(ing *networkingv1.Ingress, opts Options) (*ir.MigrationBundle, error) {
	if ing == nil {
		return nil, fmt.Errorf("ingress is nil")
	}
	if opts.Provider == "" {
		opts.Provider = ir.ProviderStandard
	}

	ns := ing.Namespace
	if ns == "" {
		ns = "default"
	}
	gwName := opts.GatewayName
	if gwName == "" {
		gwName = ing.Name + "-gateway"
	}

	meta := nginx.AuditMeta{IngressName: ing.Name, Namespace: ns}
	ann := nginx.Translate(ing.Annotations, opts.Provider, meta)

	bundle := &ir.MigrationBundle{
		SourceNamespace: ns,
		Findings:        append([]ir.AuditFinding{}, ann.Findings...),
		Policies:        append([]ir.PolicyIR{}, ann.Policies...),
		Certificates:    append([]ir.CertificateIR{}, ann.Certificates...),
	}

	// Always record core path/service translation as direct.
	bundle.Findings = append(bundle.Findings,
		ir.NewFinding(ir.FindingIDSpecRules, ir.StatusDirect, 1, "spec.rules",
			"Ingress rules mapped to HTTPRoute matches and backendRefs").
			WithTarget("HTTPRoute.spec.rules").
			WithEvidence(ir.Evidence{IngressName: ing.Name, Namespace: ns}),
	)

	hostnames := collectHostnames(ing)
	if ann.SSLPassthrough && ann.TLS != nil {
		ann.TLS.Mode = "Passthrough"
	} else if ann.SSLPassthrough {
		ann.TLS = &ir.TLSIR{Mode: "Passthrough"}
	}
	listeners := buildListeners(ing, hostnames, ann.TLS, ann.SSLPassthrough)
	if opts.IncludeGateway {
		bundle.Gateways = append(bundle.Gateways, ir.GatewayIR{
			Name:      gwName,
			Namespace: ns,
			ClassName: opts.GatewayClass,
			Listeners: listeners,
		})
	}

	route := ir.HTTPRouteIR{
		Name:        ing.Name,
		Namespace:   ns,
		Hostnames:   hostnames,
		ParentRefs:  []ir.ParentRefIR{{Name: gwName, Namespace: ns}},
		Annotations: map[string]string{"gateshift.io/source-ingress": ing.Name},
	}

	routeFilters, redirectFilters := splitRedirectFilters(ann.Filters)
	// Gateway API: RequestRedirect rules must not include backendRefs.
	for _, rf := range redirectFilters {
		route.Rules = append(route.Rules, ir.HTTPRouteRuleIR{
			Matches: []ir.HTTPMatchIR{{PathType: "PathPrefix", PathValue: "/"}},
			Filters: []ir.FilterIR{rf},
		})
	}

	if ing.Spec.DefaultBackend != nil {
		route.Rules = append(route.Rules, ruleFromBackend(ing.Spec.DefaultBackend, "/", "PathPrefix", routeFilters))
	} else if ann.DefaultBackend != "" {
		route.Rules = append(route.Rules, ruleFromAnnotatedBackend(ann.DefaultBackend, ns, routeFilters))
	}

	localRegex := isTruthyAnnotation(ing.Annotations[nginxquirks.AnnUseRegex]) ||
		strings.TrimSpace(ing.Annotations[nginxquirks.AnnRewriteTarget]) != ""

	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		host := rule.Host
		if host == "" {
			host = "*"
		}
		hostRegex := localRegex || opts.quirkHostRegex[host]
		for _, path := range rule.HTTP.Paths {
			pathValue := path.Path
			if pathValue == "" {
				pathValue = "/"
			}
			pathType := "PathPrefix"
			if path.PathType != nil {
				switch *path.PathType {
				case networkingv1.PathTypeExact:
					pathType = "Exact"
				case networkingv1.PathTypePrefix:
					pathType = "PathPrefix"
				case networkingv1.PathTypeImplementationSpecific:
					if hostRegex {
						pathType = "RegularExpression"
						bundle.Findings = append(bundle.Findings,
							ir.NewFinding(ir.FindingIDPathTypeRegex, ir.StatusRequiresPolicy, 2, "pathType",
								"ImplementationSpecific + Ingress-NGINX regex mode → RegularExpression (Extended feature)").
								WithValue(string(*path.PathType)).
								WithTarget("HTTPRoute.spec.rules[].matches[].path.type=RegularExpression").
								WithEvidence(ir.Evidence{IngressName: ing.Name, Namespace: ns, Host: host, Path: pathValue}),
						)
					} else {
						pathType = "PathPrefix"
						bundle.Findings = append(bundle.Findings,
							ir.NewFinding(ir.FindingIDPathTypeImplSpecific, ir.StatusRequiresPolicy, 2, "pathType",
								"ImplementationSpecific pathType lowered to PathPrefix; verify regex intent manually").
								WithValue(string(*path.PathType)).
								WithTarget("HTTPRoute.spec.rules[].matches[].path.type").
								WithEvidence(ir.Evidence{IngressName: ing.Name, Namespace: ns, Host: host, Path: pathValue}),
						)
					}
				}
			} else if hostRegex {
				pathType = "RegularExpression"
			}

			// Preserve Ingress-NGINX case-insensitive prefix regex semantics.
			if opts.PreserveNGINXRegex && hostRegex {
				pathValue = nginxquirks.PreserveRegexPath(pathValue)
				pathType = "RegularExpression"
				bundle.Findings = append(bundle.Findings,
					ir.NewFinding(ir.FindingIDQuirkPreserveRegex, ir.StatusDirect, 1,
						"gateshift.io/nginx-quirk/preserve-regex",
						"Emitted case-insensitive prefix RegularExpression to approximate Ingress-NGINX regex semantics").
						WithValue(pathValue).
						WithTarget("HTTPRoute.spec.rules[].matches[].path").
						WithEvidence(ir.Evidence{IngressName: ing.Name, Namespace: ns, Host: host, Path: pathValue}).
						WithFix("--preserve-nginx-regex"),
				)
			}

			// Optional trailing-slash 301 preservation.
			if opts.EmitTrailingSlashRedirects {
				if without, with, ok := nginxquirks.TrailingSlashRedirect(path.Path, path.PathType); ok {
					code := 301
					route.Rules = append(route.Rules, ir.HTTPRouteRuleIR{
						Matches: []ir.HTTPMatchIR{{PathType: "Exact", PathValue: without}},
						Filters: []ir.FilterIR{{
							Kind:               ir.FilterRequestRedirect,
							RedirectStatusCode: &code,
							RedirectPath:       strPtrLocal(with),
							RedirectPathType:   "ReplaceFullPath",
						}},
					})
					bundle.Findings = append(bundle.Findings,
						ir.NewFinding(ir.FindingIDQuirkTrailingSlashEmit, ir.StatusDirect, 1,
							"gateshift.io/nginx-quirk/trailing-slash-emit",
							"Emitted 301 trailing-slash redirect to preserve Ingress-NGINX behavior").
							WithValue(without+"→"+with).
							WithTarget("HTTPRoute RequestRedirect").
							WithEvidence(ir.Evidence{IngressName: ing.Name, Namespace: ns, Host: host, Path: path.Path}).
							WithFix("--emit-trailing-slash-redirects"),
					)
				}
			}

			be := path.Backend
			route.Rules = append(route.Rules, ruleFromBackend(&be, pathValue, pathType, routeFilters))
		}
	}

	// Attach cert DNS names when present.
	for i := range bundle.Certificates {
		bundle.Certificates[i].DNSNames = hostnames
		if bundle.Certificates[i].SecretName == "" && len(ing.Spec.TLS) > 0 {
			bundle.Certificates[i].SecretName = ing.Spec.TLS[0].SecretName
		}
	}
	if ann.TLS != nil && len(bundle.Gateways) > 0 {
		for i := range bundle.Gateways[0].Listeners {
			if bundle.Gateways[0].Listeners[i].Protocol == "HTTPS" {
				tls := *ann.TLS
				if tls.SecretName == "" && len(ing.Spec.TLS) > 0 {
					tls.SecretName = ing.Spec.TLS[0].SecretName
				}
				bundle.Gateways[0].Listeners[i].TLS = &tls
			}
		}
	}

	// Fill concrete www hostname for from-to-www-redirect placeholders.
	for i := range route.Rules {
		for j := range route.Rules[i].Filters {
			f := &route.Rules[i].Filters[j]
			if f.Kind == ir.FilterRequestRedirect && f.RedirectHostname != nil && *f.RedirectHostname == "www." {
				if len(hostnames) > 0 {
					h := hostnames[0]
					if strings.HasPrefix(h, "www.") {
						h = strings.TrimPrefix(h, "www.")
					} else {
						h = "www." + h
					}
					f.RedirectHostname = &h
				}
			}
		}
	}

	bundle.HTTPRoutes = append(bundle.HTTPRoutes, route)
	return bundle, nil
}

func strPtrLocal(s string) *string { return &s }

func isTruthyAnnotation(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}

func splitRedirectFilters(filters []ir.FilterIR) (routeFilters, redirectFilters []ir.FilterIR) {
	for _, f := range filters {
		if f.Kind == ir.FilterRequestRedirect {
			redirectFilters = append(redirectFilters, f)
			continue
		}
		routeFilters = append(routeFilters, f)
	}
	return routeFilters, redirectFilters
}

func ruleFromBackend(be *networkingv1.IngressBackend, pathValue, pathType string, filters []ir.FilterIR) ir.HTTPRouteRuleIR {
	rule := ir.HTTPRouteRuleIR{
		Matches: []ir.HTTPMatchIR{{PathType: pathType, PathValue: pathValue}},
		Filters: append([]ir.FilterIR{}, filters...),
	}
	if be != nil && be.Service != nil {
		port := int32(80)
		if be.Service.Port.Number != 0 {
			port = be.Service.Port.Number
		}
		rule.Backends = []ir.BackendRefIR{{
			Name: be.Service.Name,
			Port: port,
		}}
	}
	return rule
}

// ruleFromAnnotatedBackend parses nginx default-backend annotation (name or ns/name).
func ruleFromAnnotatedBackend(ref, defaultNS string, filters []ir.FilterIR) ir.HTTPRouteRuleIR {
	name := ref
	ns := defaultNS
	if i := strings.Index(ref, "/"); i >= 0 {
		ns = ref[:i]
		name = ref[i+1:]
	}
	return ir.HTTPRouteRuleIR{
		Matches: []ir.HTTPMatchIR{{PathType: "PathPrefix", PathValue: "/"}},
		Filters: append([]ir.FilterIR{}, filters...),
		Backends: []ir.BackendRefIR{{
			Name:      name,
			Namespace: ns,
			Port:      80,
		}},
	}
}

func collectHostnames(ing *networkingv1.Ingress) []string {
	seen := map[string]struct{}{}
	var hosts []string
	for _, rule := range ing.Spec.Rules {
		if rule.Host == "" {
			continue
		}
		if _, ok := seen[rule.Host]; ok {
			continue
		}
		seen[rule.Host] = struct{}{}
		hosts = append(hosts, rule.Host)
	}
	for _, tls := range ing.Spec.TLS {
		for _, h := range tls.Hosts {
			if _, ok := seen[h]; ok {
				continue
			}
			seen[h] = struct{}{}
			hosts = append(hosts, h)
		}
	}
	return hosts
}

func buildListeners(ing *networkingv1.Ingress, hostnames []string, tls *ir.TLSIR, passthrough bool) []ir.ListenerIR {
	hasTLS := len(ing.Spec.TLS) > 0 || tls != nil || passthrough
	tlsMode := "Terminate"
	if passthrough || (tls != nil && strings.EqualFold(tls.Mode, "Passthrough")) {
		tlsMode = "Passthrough"
	}
	var listeners []ir.ListenerIR
	if !hasTLS {
		host := ""
		if len(hostnames) > 0 {
			host = hostnames[0]
		}
		listeners = append(listeners, ir.ListenerIR{
			Name:     "http",
			Protocol: "HTTP",
			Port:     80,
			Hostname: host,
		})
		return listeners
	}

	// HTTP listener for redirects + HTTPS listeners per TLS host.
	listeners = append(listeners, ir.ListenerIR{
		Name:     "http",
		Protocol: "HTTP",
		Port:     80,
	})
	if len(ing.Spec.TLS) == 0 {
		host := ""
		if len(hostnames) > 0 {
			host = hostnames[0]
		}
		secret := ""
		if tls != nil && tlsMode != "Passthrough" {
			secret = tls.SecretName
		}
		listeners = append(listeners, ir.ListenerIR{
			Name:     "https",
			Protocol: "HTTPS",
			Port:     443,
			Hostname: host,
			TLS:      &ir.TLSIR{SecretName: secret, Mode: tlsMode},
		})
		return listeners
	}
	for i, t := range ing.Spec.TLS {
		host := ""
		if len(t.Hosts) > 0 {
			host = t.Hosts[0]
		} else if len(hostnames) > 0 {
			host = hostnames[0]
		}
		name := fmt.Sprintf("https-%d", i)
		secret := t.SecretName
		if tlsMode == "Passthrough" {
			secret = ""
		}
		listeners = append(listeners, ir.ListenerIR{
			Name:     name,
			Protocol: "HTTPS",
			Port:     443,
			Hostname: host,
			TLS: &ir.TLSIR{
				SecretName: secret,
				Mode:       tlsMode,
			},
		})
	}
	return listeners
}

// EmitYAML renders a MigrationBundle as multi-document YAML.
func EmitYAML(bundle *ir.MigrationBundle) ([]byte, error) {
	var docs []string

	for _, gw := range bundle.Gateways {
		obj, err := toGateway(gw)
		if err != nil {
			return nil, err
		}
		b, err := yaml.Marshal(obj)
		if err != nil {
			return nil, err
		}
		docs = append(docs, strings.TrimSpace(string(b)))
	}
	for _, route := range bundle.HTTPRoutes {
		obj, err := toHTTPRoute(route)
		if err != nil {
			return nil, err
		}
		b, err := yaml.Marshal(obj)
		if err != nil {
			return nil, err
		}
		docs = append(docs, strings.TrimSpace(string(b)))
	}
	for _, pol := range bundle.Policies {
		obj := toPolicyUnstructured(pol)
		b, err := yaml.Marshal(obj)
		if err != nil {
			return nil, err
		}
		docs = append(docs, strings.TrimSpace(string(b)))
	}
	for _, cert := range bundle.Certificates {
		obj := toCertificate(cert)
		b, err := yaml.Marshal(obj)
		if err != nil {
			return nil, err
		}
		docs = append(docs, strings.TrimSpace(string(b)))
	}

	return []byte(strings.Join(docs, "\n---\n") + "\n"), nil
}

func toGateway(gw ir.GatewayIR) (*gatewayv1.Gateway, error) {
	class := gatewayv1.ObjectName(gw.ClassName)
	out := &gatewayv1.Gateway{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gatewayv1.GroupVersion.String(),
			Kind:       "Gateway",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      gw.Name,
			Namespace: gw.Namespace,
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: class,
		},
	}
	for _, l := range gw.Listeners {
		listener := gatewayv1.Listener{
			Name:     gatewayv1.SectionName(l.Name),
			Protocol: gatewayv1.ProtocolType(l.Protocol),
			Port:     gatewayv1.PortNumber(l.Port),
		}
		if l.Hostname != "" {
			h := gatewayv1.Hostname(l.Hostname)
			listener.Hostname = &h
		}
		if l.TLS != nil && l.Protocol == "HTTPS" {
			mode := gatewayv1.TLSModeTerminate
			if strings.EqualFold(l.TLS.Mode, "Passthrough") {
				mode = gatewayv1.TLSModePassthrough
			}
			listener.TLS = &gatewayv1.GatewayTLSConfig{
				Mode: &mode,
			}
			// CertificateRefs are only valid for Terminate mode.
			if mode == gatewayv1.TLSModeTerminate && l.TLS.SecretName != "" {
				ns := gatewayv1.Namespace(gw.Namespace)
				listener.TLS.CertificateRefs = []gatewayv1.SecretObjectReference{{
					Name:      gatewayv1.ObjectName(l.TLS.SecretName),
					Namespace: &ns,
					Kind:      kindPtr("Secret"),
					Group:     groupPtr(corev1.GroupName),
				}}
			}
		}
		out.Spec.Listeners = append(out.Spec.Listeners, listener)
	}
	return out, nil
}

func toHTTPRoute(route ir.HTTPRouteIR) (*gatewayv1.HTTPRoute, error) {
	out := &gatewayv1.HTTPRoute{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gatewayv1.GroupVersion.String(),
			Kind:       "HTTPRoute",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        route.Name,
			Namespace:   route.Namespace,
			Annotations: route.Annotations,
		},
	}
	for _, h := range route.Hostnames {
		out.Spec.Hostnames = append(out.Spec.Hostnames, gatewayv1.Hostname(h))
	}
	for _, p := range route.ParentRefs {
		ref := gatewayv1.ParentReference{
			Name: gatewayv1.ObjectName(p.Name),
		}
		if p.Namespace != "" {
			ns := gatewayv1.Namespace(p.Namespace)
			ref.Namespace = &ns
		}
		out.Spec.ParentRefs = append(out.Spec.ParentRefs, ref)
	}
	for _, rule := range route.Rules {
		hr := gatewayv1.HTTPRouteRule{}
		for _, m := range rule.Matches {
			match := gatewayv1.HTTPRouteMatch{}
			pt := gatewayv1.PathMatchType(m.PathType)
			val := m.PathValue
			match.Path = &gatewayv1.HTTPPathMatch{Type: &pt, Value: &val}
			hr.Matches = append(hr.Matches, match)
		}
		for _, f := range rule.Filters {
			hf, ok := toHTTPRouteFilter(f)
			if ok {
				hr.Filters = append(hr.Filters, hf)
			}
		}
		for _, b := range rule.Backends {
			br := gatewayv1.HTTPBackendRef{
				BackendRef: gatewayv1.BackendRef{
					BackendObjectReference: gatewayv1.BackendObjectReference{
						Name: gatewayv1.ObjectName(b.Name),
						Port: portPtr(b.Port),
					},
				},
			}
			if b.Namespace != "" {
				ns := gatewayv1.Namespace(b.Namespace)
				br.Namespace = &ns
			}
			if b.Weight != nil {
				w := *b.Weight
				br.Weight = &w
			}
			hr.BackendRefs = append(hr.BackendRefs, br)
		}
		out.Spec.Rules = append(out.Spec.Rules, hr)
	}
	return out, nil
}

func toHTTPRouteFilter(f ir.FilterIR) (gatewayv1.HTTPRouteFilter, bool) {
	switch f.Kind {
	case ir.FilterURLRewrite:
		filter := gatewayv1.HTTPRouteFilter{Type: gatewayv1.HTTPRouteFilterURLRewrite}
		rw := &gatewayv1.HTTPURLRewriteFilter{}
		if f.ReplacePrefixMatch != nil {
			t := gatewayv1.PrefixMatchHTTPPathModifier
			rw.Path = &gatewayv1.HTTPPathModifier{
				Type:               t,
				ReplacePrefixMatch: f.ReplacePrefixMatch,
			}
		}
		if f.ReplaceFullPath != nil {
			t := gatewayv1.FullPathHTTPPathModifier
			rw.Path = &gatewayv1.HTTPPathModifier{
				Type:            t,
				ReplaceFullPath: f.ReplaceFullPath,
			}
		}
		if f.Hostname != nil {
			h := gatewayv1.PreciseHostname(*f.Hostname)
			rw.Hostname = &h
		}
		filter.URLRewrite = rw
		return filter, true
	case ir.FilterRequestRedirect:
		filter := gatewayv1.HTTPRouteFilter{Type: gatewayv1.HTTPRouteFilterRequestRedirect}
		rd := &gatewayv1.HTTPRequestRedirectFilter{}
		if f.RedirectScheme != nil {
			rd.Scheme = f.RedirectScheme
		}
		if f.RedirectStatusCode != nil {
			rd.StatusCode = f.RedirectStatusCode
		}
		if f.RedirectHostname != nil {
			h := gatewayv1.PreciseHostname(*f.RedirectHostname)
			rd.Hostname = &h
		}
		if f.RedirectPath != nil {
			t := gatewayv1.FullPathHTTPPathModifier
			if f.RedirectPathType == "ReplacePrefixMatch" {
				t = gatewayv1.PrefixMatchHTTPPathModifier
				rd.Path = &gatewayv1.HTTPPathModifier{Type: t, ReplacePrefixMatch: f.RedirectPath}
			} else {
				rd.Path = &gatewayv1.HTTPPathModifier{Type: t, ReplaceFullPath: f.RedirectPath}
			}
		}
		filter.RequestRedirect = rd
		return filter, true
	case ir.FilterExtensionRef:
		// RequestMirror and other extensions are emitted as ExtensionRef placeholders.
		if f.ExtensionKind == "RequestMirror" {
			filter := gatewayv1.HTTPRouteFilter{Type: gatewayv1.HTTPRouteFilterRequestMirror}
			filter.RequestMirror = &gatewayv1.HTTPRequestMirrorFilter{
				BackendRef: gatewayv1.BackendObjectReference{
					Name: gatewayv1.ObjectName(f.ExtensionName),
				},
			}
			return filter, true
		}
		return gatewayv1.HTTPRouteFilter{}, false
	case ir.FilterResponseHeader:
		filter := gatewayv1.HTTPRouteFilter{Type: gatewayv1.HTTPRouteFilterResponseHeaderModifier}
		mod := &gatewayv1.HTTPHeaderFilter{}
		for k, v := range f.SetHeaders {
			mod.Set = append(mod.Set, gatewayv1.HTTPHeader{Name: gatewayv1.HTTPHeaderName(k), Value: v})
		}
		for k, v := range f.AddHeaders {
			mod.Add = append(mod.Add, gatewayv1.HTTPHeader{Name: gatewayv1.HTTPHeaderName(k), Value: v})
		}
		for _, h := range f.RemoveHeaders {
			mod.Remove = append(mod.Remove, h)
		}
		filter.ResponseHeaderModifier = mod
		return filter, true
	case ir.FilterRequestHeader:
		filter := gatewayv1.HTTPRouteFilter{Type: gatewayv1.HTTPRouteFilterRequestHeaderModifier}
		mod := &gatewayv1.HTTPHeaderFilter{}
		for k, v := range f.SetHeaders {
			mod.Set = append(mod.Set, gatewayv1.HTTPHeader{Name: gatewayv1.HTTPHeaderName(k), Value: v})
		}
		filter.RequestHeaderModifier = mod
		return filter, true
	default:
		return gatewayv1.HTTPRouteFilter{}, false
	}
}

func toPolicyUnstructured(pol ir.PolicyIR) map[string]any {
	apiVersion, _ := pol.Spec["apiVersion"].(string)
	kind, _ := pol.Spec["kind"].(string)
	if apiVersion == "" {
		apiVersion = "gateshift.io/v1alpha1"
	}
	if kind == "" {
		kind = "RateLimitPolicy"
	}
	spec := map[string]any{}
	for k, v := range pol.Spec {
		if k == "apiVersion" || k == "kind" {
			continue
		}
		spec[k] = v
	}
	spec["targetRef"] = map[string]any{
		"group":     "gateway.networking.k8s.io",
		"kind":      "HTTPRoute",
		"name":      pol.TargetRef.Name,
		"namespace": pol.TargetRef.Namespace,
	}
	return map[string]any{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata": map[string]any{
			"name":      pol.Name,
			"namespace": pol.Namespace,
			"labels": map[string]string{
				"gateshift.io/policy": string(pol.Kind),
				"gateshift.io/target": string(pol.Provider),
			},
		},
		"spec": spec,
	}
}

func toCertificate(cert ir.CertificateIR) map[string]any {
	issuerRef := map[string]any{
		"kind": "ClusterIssuer",
		"name": cert.ClusterIssuer,
	}
	if cert.ClusterIssuer == "" && cert.Issuer != "" {
		issuerRef = map[string]any{
			"kind": "Issuer",
			"name": cert.Issuer,
		}
	}
	return map[string]any{
		"apiVersion": "cert-manager.io/v1",
		"kind":       "Certificate",
		"metadata": map[string]any{
			"name":      cert.Name,
			"namespace": cert.Namespace,
		},
		"spec": map[string]any{
			"secretName": cert.SecretName,
			"dnsNames":   cert.DNSNames,
			"issuerRef":  issuerRef,
		},
	}
}

func kindPtr(s string) *gatewayv1.Kind {
	k := gatewayv1.Kind(s)
	return &k
}

func groupPtr(s string) *gatewayv1.Group {
	g := gatewayv1.Group(s)
	return &g
}

func portPtr(p int32) *gatewayv1.PortNumber {
	pn := gatewayv1.PortNumber(p)
	return &pn
}
