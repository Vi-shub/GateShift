// Package ir defines GateShift's Intermediate Representation used between
// Ingress/annotation parsers and Gateway API emitters.
package ir

import (
	"fmt"
	"strings"
)

// Status classifies how an annotation or rule translates.
type Status string

const (
	StatusDirect         Status = "direct"          // 🟢 1:1 Gateway API mapping
	StatusRequiresPolicy Status = "requires_policy" // 🟡 needs vendor extension / Policy CRD
	StatusUntranslatable Status = "untranslatable"  // 🔴 needs manual intervention
)

// Provider identifies the target Gateway API implementation.
type Provider string

const (
	ProviderStandard     Provider = "standard"
	ProviderEnvoyGateway Provider = "envoy-gateway"
	ProviderCilium       Provider = "cilium"
	ProviderIstio        Provider = "istio"
	ProviderKong         Provider = "kong"
)

// ParseProvider normalizes a CLI/provider string.
func ParseProvider(s string) (Provider, error) {
	p := Provider(strings.ToLower(strings.TrimSpace(s)))
	switch p {
	case ProviderStandard, ProviderEnvoyGateway, ProviderCilium, ProviderIstio, ProviderKong, "":
		if p == "" {
			return ProviderStandard, nil
		}
		return p, nil
	default:
		return "", fmt.Errorf("unsupported target provider %q (use standard|envoy-gateway|cilium|istio|kong)", s)
	}
}

// AuditFinding records how a single annotation or feature was classified.
type AuditFinding struct {
	Key         string `json:"key"`
	Value       string `json:"value,omitempty"`
	Status      Status `json:"status"`
	Level       int    `json:"level,omitempty"` // 1=direct, 2=policy CRD, 3=untranslatable
	Target      string `json:"target,omitempty"`
	Message     string `json:"message"`
	IngressName string `json:"ingressName,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
}

// MigrationBundle is the IR produced from one or more Ingress resources.
type MigrationBundle struct {
	SourceNamespace string           `json:"sourceNamespace"`
	Gateways        []GatewayIR      `json:"gateways,omitempty"`
	HTTPRoutes      []HTTPRouteIR    `json:"httpRoutes,omitempty"`
	Policies        []PolicyIR       `json:"policies,omitempty"`
	Certificates    []CertificateIR  `json:"certificates,omitempty"`
	Findings        []AuditFinding   `json:"findings,omitempty"`
}

// GatewayIR is a provider-neutral Gateway description.
type GatewayIR struct {
	Name      string
	Namespace string
	ClassName string
	Listeners []ListenerIR
}

// ListenerIR describes a Gateway listener.
type ListenerIR struct {
	Name     string
	Protocol string // HTTP | HTTPS
	Port     int32
	Hostname string
	TLS      *TLSIR
}

// TLSIR holds TLS termination settings derived from Ingress TLS / cert-manager.
type TLSIR struct {
	SecretName    string
	ClusterIssuer string
	Issuer        string
	Mode          string // Terminate
}

// HTTPRouteIR is a provider-neutral HTTPRoute description.
type HTTPRouteIR struct {
	Name        string
	Namespace   string
	Hostnames   []string
	ParentRefs  []ParentRefIR
	Rules       []HTTPRouteRuleIR
	Annotations map[string]string
}

// ParentRefIR links an HTTPRoute to a Gateway.
type ParentRefIR struct {
	Name      string
	Namespace string
	Section   string
}

// HTTPRouteRuleIR is one routing rule with matches, filters, and backends.
type HTTPRouteRuleIR struct {
	Matches  []HTTPMatchIR
	Filters  []FilterIR
	Backends []BackendRefIR
}

// HTTPMatchIR describes path/method/header matching.
type HTTPMatchIR struct {
	PathType  string // Exact | PathPrefix | RegularExpression
	PathValue string
	Method    string
	Headers   map[string]string
}

// FilterKind identifies a Gateway API filter or GateShift policy intent.
type FilterKind string

const (
	FilterURLRewrite      FilterKind = "URLRewrite"
	FilterRequestRedirect FilterKind = "RequestRedirect"
	FilterRequestHeader   FilterKind = "RequestHeaderModifier"
	FilterResponseHeader  FilterKind = "ResponseHeaderModifier"
	FilterExtensionRef    FilterKind = "ExtensionRef"
)

// FilterIR is a structured filter intent produced by adapters.
type FilterIR struct {
	Kind FilterKind

	// URLRewrite
	ReplacePrefixMatch *string
	ReplaceFullPath    *string
	Hostname           *string

	// RequestRedirect
	RedirectScheme     *string
	RedirectHostname   *string
	RedirectPath       *string
	RedirectPort       *int32
	RedirectStatusCode *int
	RedirectPathType   string // ReplaceFullPath | ReplacePrefixMatch

	// Header modifiers
	SetHeaders    map[string]string
	AddHeaders    map[string]string
	RemoveHeaders []string

	// Extension / policy
	ExtensionGroup string
	ExtensionKind  string
	ExtensionName  string
}

// BackendRefIR points at a Service backend.
type BackendRefIR struct {
	Name      string
	Namespace string
	Port      int32
	Weight    *int32
}

// PolicyKind identifies vendor-specific policy artifacts.
type PolicyKind string

const (
	PolicyRateLimit       PolicyKind = "RateLimit"
	PolicyCORS            PolicyKind = "CORS"
	PolicyTLSCert         PolicyKind = "TLSCertificate"
	PolicySessionAffinity PolicyKind = "SessionAffinity"
	PolicyIPFilter        PolicyKind = "IPFilter"
	PolicyBackendTuning   PolicyKind = "BackendTuning"
)

// PolicyIR is an implementation-specific extension GateShift may emit.
type PolicyIR struct {
	Kind      PolicyKind
	Name      string
	Namespace string
	TargetRef ParentRefIR
	Provider  Provider
	Spec      map[string]any
}

// CertificateIR maps cert-manager style issuer bindings.
type CertificateIR struct {
	Name          string
	Namespace     string
	SecretName    string
	DNSNames      []string
	ClusterIssuer string
	Issuer        string
}

// Summary returns counts by audit status.
func (b *MigrationBundle) Summary() (direct, policy, untranslatable int) {
	for _, f := range b.Findings {
		switch f.Status {
		case StatusDirect:
			direct++
		case StatusRequiresPolicy:
			policy++
		case StatusUntranslatable:
			untranslatable++
		}
	}
	return
}
