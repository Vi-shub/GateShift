package nginx

import (
	"strings"

	"github.com/gateshift/gateshift/pkg/adapters"
	"github.com/gateshift/gateshift/pkg/ir"
)

// CertManagerAdapter binds cert-manager issuer annotations to Certificate + Gateway TLS (Level 2).
type CertManagerAdapter struct{}

func (CertManagerAdapter) Name() string          { return "cert-manager" }
func (CertManagerAdapter) Level() adapters.Level { return adapters.Level2 }
func (CertManagerAdapter) CanHandle(key string) bool {
	return key == AnnCertManagerClusterIssuer || key == AnnCertManagerIssuer
}

func (CertManagerAdapter) Transform(key, value string, ctx *adapters.Context) error {
	value = strings.TrimSpace(value)
	if ctx.TLS == nil {
		ctx.TLS = &ir.TLSIR{Mode: "Terminate"}
	}
	cert := ir.CertificateIR{
		Name:       ctx.Meta.IngressName + "-cert",
		Namespace:  ctx.Meta.Namespace,
		SecretName: ctx.Meta.IngressName + "-tls",
	}
	if key == AnnCertManagerClusterIssuer {
		ctx.TLS.ClusterIssuer = value
		cert.ClusterIssuer = value
	} else {
		ctx.TLS.Issuer = value
		cert.Issuer = value
	}
	ctx.Certificates = append(ctx.Certificates, cert)
	ctx.AddFinding(key, value, ir.StatusRequiresPolicy, adapters.Level2,
		"Certificate/Gateway.listeners[].tls",
		"cert-manager issuer mapped to Certificate CR + Gateway TLS binding")
	return nil
}

// AffinityAdapter maps session affinity + session-cookie-* family → Level 2 policy.
type AffinityAdapter struct{}

func (AffinityAdapter) Name() string          { return "affinity" }
func (AffinityAdapter) Level() adapters.Level { return adapters.Level2 }
func (AffinityAdapter) CanHandle(key string) bool {
	switch key {
	case AnnAffinity, AnnSessionCookieName, AnnSessionCookieExpires, AnnSessionCookieMaxAge,
		AnnSessionCookieSecure, AnnSessionCookieSameSite, AnnSessionCookieConditionalSameSiteNone,
		AnnSessionCookiePath, AnnSessionCookieChangeOnFailure, AnnSessionCookieHash:
		return true
	default:
		return false
	}
}

func affinitySiblingKeys() []string {
	return []string{
		AnnAffinity,
		AnnSessionCookieName,
		AnnSessionCookieExpires,
		AnnSessionCookieMaxAge,
		AnnSessionCookieSecure,
		AnnSessionCookieSameSite,
		AnnSessionCookieConditionalSameSiteNone,
		AnnSessionCookiePath,
		AnnSessionCookieChangeOnFailure,
		AnnSessionCookieHash,
	}
}

func (AffinityAdapter) Transform(key, value string, ctx *adapters.Context) error {
	// Run once for the whole cookie-affinity family.
	for _, s := range affinitySiblingKeys() {
		if s == key {
			continue
		}
		if ctx.Claimed[s] {
			ctx.Claim(key)
			return nil
		}
	}

	cookie := ctx.Annotations[AnnSessionCookieName]
	if cookie == "" {
		cookie = "INGRESSCOOKIE"
	}

	expires := strings.TrimSpace(ctx.Annotations[AnnSessionCookieExpires])
	maxAge := strings.TrimSpace(ctx.Annotations[AnnSessionCookieMaxAge])
	sameSite := strings.TrimSpace(ctx.Annotations[AnnSessionCookieSameSite])
	secure := ctx.Annotations[AnnSessionCookieSecure]
	conditionalNone := ctx.Annotations[AnnSessionCookieConditionalSameSiteNone]
	path := strings.TrimSpace(ctx.Annotations[AnnSessionCookiePath])
	changeOnFailure := ctx.Annotations[AnnSessionCookieChangeOnFailure]
	hash := strings.TrimSpace(ctx.Annotations[AnnSessionCookieHash])

	timeout := "3600s"
	if maxAge != "" {
		timeout = maxAge + "s"
	} else if expires != "" {
		timeout = expires + "s"
	}

	pol := ir.PolicyIR{
		Kind:      ir.PolicySessionAffinity,
		Name:      ctx.Meta.IngressName + "-affinity",
		Namespace: ctx.Meta.Namespace,
		Provider:  ctx.Provider,
		TargetRef: ir.ParentRefIR{Name: ctx.Meta.IngressName, Namespace: ctx.Meta.Namespace},
		Spec: map[string]any{
			"apiVersion":  "gateshift.io/v1alpha1",
			"kind":        "SessionAffinityPolicy",
			"affinity":    ctx.Annotations[AnnAffinity],
			"cookieName":  cookie,
			"featureGate": "SessionPersistence",
		},
	}
	if expires != "" {
		pol.Spec["cookieExpires"] = expires
	}
	if maxAge != "" {
		pol.Spec["cookieMaxAge"] = maxAge
	}
	if sameSite != "" {
		pol.Spec["cookieSameSite"] = sameSite
	}
	if secure != "" {
		pol.Spec["cookieSecure"] = isTruthy(secure)
	}
	if conditionalNone != "" {
		pol.Spec["cookieConditionalSameSiteNone"] = isTruthy(conditionalNone)
	}
	if path != "" {
		pol.Spec["cookiePath"] = path
	}
	if changeOnFailure != "" {
		pol.Spec["cookieChangeOnFailure"] = isTruthy(changeOnFailure)
	}
	if hash != "" {
		pol.Spec["cookieHash"] = hash
	}

	if ctx.Provider == ir.ProviderEnvoyGateway {
		// Envoy Gateway BackendTrafficPolicy has no sessionPersistence field.
		// Cookie stickiness maps to loadBalancer.consistentHash (EG v1.2+).
		cookieSpec := map[string]any{
			"name": cookie,
			"ttl":  timeout,
		}
		attrs := map[string]string{}
		if sameSite != "" {
			attrs["SameSite"] = sameSite
		}
		if secure != "" {
			if isTruthy(secure) {
				attrs["Secure"] = "true"
			} else {
				attrs["Secure"] = "false"
			}
		}
		if path != "" {
			attrs["Path"] = path
		}
		if len(attrs) > 0 {
			cookieSpec["attributes"] = attrs
		}
		pol.Spec = map[string]any{
			"apiVersion": "gateway.envoyproxy.io/v1alpha1",
			"kind":       "BackendTrafficPolicy",
			"loadBalancer": map[string]any{
				"type": "ConsistentHash",
				"consistentHash": map[string]any{
					"type":   "Cookie",
					"cookie": cookieSpec,
				},
			},
		}
	}

	ctx.Policies = append(ctx.Policies, pol)
	ctx.AddFinding(AnnAffinity, ctx.Annotations[AnnAffinity], ir.StatusRequiresPolicy, adapters.Level2,
		"SessionPersistence / vendor policy",
		"Session affinity + cookie attributes mapped to SessionPersistence / BackendTrafficPolicy")

	for _, sibling := range affinitySiblingKeys() {
		if sibling == AnnAffinity {
			continue
		}
		if v, ok := ctx.Annotations[sibling]; ok {
			ctx.AddFinding(sibling, v, ir.StatusRequiresPolicy, adapters.Level2,
				"SessionPersistence cookie attributes",
				"Consumed by affinity/session-cookie adapter")
		}
	}
	ctx.Claim(affinitySiblingKeys()...)
	return nil
}

// IPAllowAdapter maps whitelist/denylist-source-range → Level 2 SecurityPolicy.
type IPAllowAdapter struct{}

func (IPAllowAdapter) Name() string          { return "ip-filter" }
func (IPAllowAdapter) Level() adapters.Level { return adapters.Level2 }
func (IPAllowAdapter) CanHandle(key string) bool {
	return key == AnnWhitelistSourceRange || key == AnnDenylistSourceRange
}

func (IPAllowAdapter) Transform(key, value string, ctx *adapters.Context) error {
	cidrs := splitCSV(value)
	action := "Allow"
	if key == AnnDenylistSourceRange {
		action = "Deny"
	}
	defaultAction := "Deny"
	if action == "Deny" {
		defaultAction = "Allow"
	}
	pol := ir.PolicyIR{
		Kind:      ir.PolicyIPFilter,
		Name:      ctx.Meta.IngressName + "-ipfilter",
		Namespace: ctx.Meta.Namespace,
		Provider:  ctx.Provider,
		TargetRef: ir.ParentRefIR{Name: ctx.Meta.IngressName, Namespace: ctx.Meta.Namespace},
		Spec: map[string]any{
			"action": action,
			"cidrs":  cidrs,
		},
	}
	switch ctx.Provider {
	case ir.ProviderEnvoyGateway:
		// Only emit CRD-valid SecurityPolicy fields (do not leak action/cidrs IR keys).
		pol.Spec = map[string]any{
			"apiVersion": "gateway.envoyproxy.io/v1alpha1",
			"kind":       "SecurityPolicy",
			"authorization": map[string]any{
				"defaultAction": defaultAction,
				"rules": []any{
					map[string]any{
						"action": action,
						"principal": map[string]any{
							"clientCIDRs": cidrs,
						},
					},
				},
			},
		}
	default:
		pol.Spec["apiVersion"] = "gateshift.io/v1alpha1"
		pol.Spec["kind"] = "IPFilterPolicy"
	}
	ctx.Policies = append(ctx.Policies, pol)
	ctx.AddFinding(key, value, ir.StatusRequiresPolicy, adapters.Level2,
		"SecurityPolicy / IP allow list",
		"Source IP filtering requires a vendor SecurityPolicy CRD")
	return nil
}

// TimeoutBodyAdapter maps proxy timeouts / body / buffering / retries → BackendTrafficPolicy.
type TimeoutBodyAdapter struct{}

func (TimeoutBodyAdapter) Name() string          { return "proxy-timeouts" }
func (TimeoutBodyAdapter) Level() adapters.Level { return adapters.Level2 }
func (TimeoutBodyAdapter) CanHandle(key string) bool {
	switch key {
	case AnnProxyBodySize, AnnProxyReadTimeout, AnnProxySendTimeout, AnnBackendProtocol,
		AnnProxyConnectTimeout, AnnProxyBuffering, AnnClientBodyBufferSize, AnnProxyNextUpstream:
		return true
	default:
		return false
	}
}

func proxyTuningKeys() []string {
	return []string{
		AnnProxyBodySize, AnnProxyReadTimeout, AnnProxySendTimeout, AnnBackendProtocol,
		AnnProxyConnectTimeout, AnnProxyBuffering, AnnClientBodyBufferSize, AnnProxyNextUpstream,
	}
}

func (TimeoutBodyAdapter) Transform(key, value string, ctx *adapters.Context) error {
	for _, s := range proxyTuningKeys() {
		if s != key && ctx.Claimed[s] {
			ctx.Claim(key)
			return nil
		}
	}

	pol := ir.PolicyIR{
		Kind:      ir.PolicyBackendTuning,
		Name:      ctx.Meta.IngressName + "-backend",
		Namespace: ctx.Meta.Namespace,
		Provider:  ctx.Provider,
		TargetRef: ir.ParentRefIR{Name: ctx.Meta.IngressName, Namespace: ctx.Meta.Namespace},
		Spec: map[string]any{
			"apiVersion": "gateshift.io/v1alpha1",
			"kind":       "BackendTrafficPolicy",
		},
	}

	if ctx.Provider == ir.ProviderEnvoyGateway {
		// Emit only fields valid on gateway.envoyproxy.io/v1alpha1 BackendTrafficPolicy.
		pol.Spec = map[string]any{
			"apiVersion": "gateway.envoyproxy.io/v1alpha1",
			"kind":       "BackendTrafficPolicy",
		}
		timeout := map[string]any{}
		httpTO := map[string]any{}
		tcpTO := map[string]any{}
		if v := ctx.Annotations[AnnProxyReadTimeout]; v != "" {
			httpTO["requestTimeout"] = v + "s"
		} else if v := ctx.Annotations[AnnProxySendTimeout]; v != "" {
			httpTO["requestTimeout"] = v + "s"
		}
		if v := ctx.Annotations[AnnProxyConnectTimeout]; v != "" {
			tcpTO["connectTimeout"] = v + "s"
		}
		if len(httpTO) > 0 {
			timeout["http"] = httpTO
		}
		if len(tcpTO) > 0 {
			timeout["tcp"] = tcpTO
		}
		if len(timeout) > 0 {
			pol.Spec["timeout"] = timeout
		}
		if v := ctx.Annotations[AnnProxyBodySize]; v != "" {
			pol.Spec["connection"] = map[string]any{
				"bufferLimit": normalizeKubeQuantity(v),
			}
		}
		// Buffering / next-upstream / backend-protocol: keep as findings only (no portable EG fields).
	} else {
		if v := ctx.Annotations[AnnProxyBodySize]; v != "" {
			pol.Spec["maxRequestBodySize"] = v
		}
		if v := ctx.Annotations[AnnProxyReadTimeout]; v != "" {
			pol.Spec["readTimeout"] = v + "s"
		}
		if v := ctx.Annotations[AnnProxySendTimeout]; v != "" {
			pol.Spec["sendTimeout"] = v + "s"
		}
		if v := ctx.Annotations[AnnProxyConnectTimeout]; v != "" {
			pol.Spec["connectTimeout"] = v + "s"
		}
		if v := ctx.Annotations[AnnBackendProtocol]; v != "" {
			pol.Spec["backendProtocol"] = v
		}
		if v := ctx.Annotations[AnnProxyBuffering]; v != "" {
			pol.Spec["buffering"] = map[string]any{"enabled": isTruthy(v)}
		}
		if v := ctx.Annotations[AnnClientBodyBufferSize]; v != "" {
			pol.Spec["clientBodyBufferSize"] = v
		}
		if v := ctx.Annotations[AnnProxyNextUpstream]; v != "" {
			pol.Spec["retry"] = map[string]any{
				"nextUpstream": v,
				"note":         "Mapped from proxy-next-upstream; verify Envoy retryOn equivalents",
			}
		}
	}

	ctx.Policies = append(ctx.Policies, pol)
	for _, k := range proxyTuningKeys() {
		if v, ok := ctx.Annotations[k]; ok {
			ctx.AddFinding(k, v, ir.StatusRequiresPolicy, adapters.Level2,
				"BackendTrafficPolicy",
				"Proxy tuning maps to provider BackendTrafficPolicy fields")
		}
	}
	ctx.Claim(proxyTuningKeys()...)
	return nil
}

// normalizeKubeQuantity maps nginx-style sizes (8m, 1k) to Kubernetes Quantity strings.
func normalizeKubeQuantity(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return v
	}
	lower := strings.ToLower(v)
	switch {
	case strings.HasSuffix(lower, "mi"), strings.HasSuffix(lower, "gi"),
		strings.HasSuffix(lower, "ki"), strings.HasSuffix(lower, "ti"):
		return v
	case strings.HasSuffix(lower, "m"):
		return strings.TrimSuffix(strings.TrimSuffix(v, "m"), "M") + "Mi"
	case strings.HasSuffix(lower, "g"):
		return strings.TrimSuffix(strings.TrimSuffix(v, "g"), "G") + "Gi"
	case strings.HasSuffix(lower, "k"):
		return strings.TrimSuffix(strings.TrimSuffix(v, "k"), "K") + "Ki"
	default:
		return v
	}
}

// CanaryAdapter maps canary annotations → Level 2 weighted backend / header match notes.
type CanaryAdapter struct{}

func (CanaryAdapter) Name() string          { return "canary" }
func (CanaryAdapter) Level() adapters.Level { return adapters.Level2 }
func (CanaryAdapter) CanHandle(key string) bool {
	return key == AnnCanary || key == AnnCanaryWeight || key == AnnCanaryByHeader
}

func (CanaryAdapter) Transform(key, value string, ctx *adapters.Context) error {
	ctx.AddFinding(key, value, ir.StatusRequiresPolicy, adapters.Level2,
		"HTTPRoute.spec.rules[].backendRefs.weight / matches.headers",
		"Canary Ingresses should be merged into a single weighted HTTPRoute manually or via GateShift canary merge (planned)")
	return nil
}

// RegexAdapter flags use-regex as Level 2 Extended path match.
type RegexAdapter struct{}

func (RegexAdapter) Name() string          { return "use-regex" }
func (RegexAdapter) Level() adapters.Level { return adapters.Level2 }
func (RegexAdapter) CanHandle(key string) bool {
	return key == AnnUseRegex || key == AnnUpstreamHashBy
}

func (RegexAdapter) Transform(key, value string, ctx *adapters.Context) error {
	if key == AnnUseRegex && isTruthy(value) {
		ctx.AddFinding(key, value, ir.StatusRequiresPolicy, adapters.Level2,
			"HTTPRoute path type=RegularExpression",
			"Regex paths are Gateway API Extended. Ingress-NGINX also applies case-insensitive prefix matching host-wide — see gateshift.io/nginx-quirk/* findings and --preserve-nginx-regex")
		return nil
	}
	ctx.AddFinding(key, value, ir.StatusRequiresPolicy, adapters.Level2,
		"BackendTrafficPolicy consistentHash",
		"Upstream hash requires provider-specific load balancer policy")
	return nil
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
