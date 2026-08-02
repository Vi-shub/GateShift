package nginx

import (
	"strings"

	"github.com/gateshift/gateshift/pkg/adapters"
	"github.com/gateshift/gateshift/pkg/ir"
)

// BackendTLSAdapter maps proxy-ssl-* → BackendTLSPolicy scaffold.
type BackendTLSAdapter struct{}

func (BackendTLSAdapter) Name() string          { return "backend-tls" }
func (BackendTLSAdapter) Level() adapters.Level { return adapters.Level2 }
func (BackendTLSAdapter) CanHandle(key string) bool {
	return key == AnnProxySSLSecret || key == AnnProxySSLVerify
}

func (BackendTLSAdapter) Transform(key, value string, ctx *adapters.Context) error {
	secret := ctx.Annotations[AnnProxySSLSecret]
	verify := ctx.Annotations[AnnProxySSLVerify]
	pol := ir.PolicyIR{
		Kind:      ir.PolicyBackendTuning,
		Name:      ctx.Meta.IngressName + "-backend-tls",
		Namespace: ctx.Meta.Namespace,
		Provider:  ctx.Provider,
		TargetRef: ir.ParentRefIR{Name: ctx.Meta.IngressName, Namespace: ctx.Meta.Namespace},
		Spec: map[string]any{
			"apiVersion": "gateway.networking.k8s.io/v1alpha3",
			"kind":       "BackendTLSPolicy",
			"secretName": secret,
			"verify":     verify,
		},
	}
	ctx.Policies = append(ctx.Policies, pol)
	ctx.AddFinding(key, value, ir.StatusRequiresPolicy, adapters.Level2,
		"BackendTLSPolicy",
		"Backend TLS annotations mapped to BackendTLSPolicy scaffold")
	ctx.Claim(AnnProxySSLSecret, AnnProxySSLVerify)
	return nil
}

// MirrorAdapter maps mirror / mirror-target → RequestMirror filter intent (as Extension/policy note).
type MirrorAdapter struct{}

func (MirrorAdapter) Name() string          { return "mirror" }
func (MirrorAdapter) Level() adapters.Level { return adapters.Level2 }
func (MirrorAdapter) CanHandle(key string) bool {
	return key == AnnMirror || key == AnnMirrorTarget
}

func (MirrorAdapter) Transform(key, value string, ctx *adapters.Context) error {
	if key == AnnMirror && !isTruthy(value) {
		ctx.AddFinding(key, value, ir.StatusDirect, adapters.Level1, "", "mirror disabled")
		return nil
	}
	target := ctx.Annotations[AnnMirrorTarget]
	if target == "" {
		target = value
	}
	// Encode as ExtensionRef-style filter placeholder on IR.
	ctx.Filters = append(ctx.Filters, ir.FilterIR{
		Kind:           ir.FilterExtensionRef,
		ExtensionGroup: "gateway.networking.k8s.io",
		ExtensionKind:  "RequestMirror",
		ExtensionName:  strings.ReplaceAll(target, "/", "-"),
	})
	ctx.AddFinding(key, value, ir.StatusRequiresPolicy, adapters.Level2,
		"HTTPRoute.spec.rules[].filters[type=RequestMirror]",
		"Traffic mirroring mapped to RequestMirror filter — set backendRef to "+target)
	ctx.Claim(AnnMirror, AnnMirrorTarget)
	return nil
}
