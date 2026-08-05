package nginx

import (
	"fmt"
	"strings"

	"github.com/gateshift/gateshift/pkg/adapters"
	"github.com/gateshift/gateshift/pkg/ir"
)

// AccessLogAdapter maps enable-access-log → observability / ClientTrafficPolicy note.
type AccessLogAdapter struct{}

func (AccessLogAdapter) Name() string          { return "access-log" }
func (AccessLogAdapter) Level() adapters.Level { return adapters.Level2 }
func (AccessLogAdapter) CanHandle(key string) bool {
	return key == AnnEnableAccessLog
}

func (AccessLogAdapter) Transform(key, value string, ctx *adapters.Context) error {
	enabled := isTruthy(value)
	pol := ir.PolicyIR{
		Kind:      ir.PolicyBackendTuning,
		Name:      ctx.Meta.IngressName + "-accesslog",
		Namespace: ctx.Meta.Namespace,
		Provider:  ctx.Provider,
		TargetRef: ir.ParentRefIR{Name: ctx.Meta.IngressName, Namespace: ctx.Meta.Namespace},
		Spec: map[string]any{
			"apiVersion": "gateshift.io/v1alpha1",
			"kind":       "AccessLogPolicy",
			"enabled":    enabled,
		},
	}
	if ctx.Provider == ir.ProviderEnvoyGateway {
		pol.Spec["apiVersion"] = "gateway.envoyproxy.io/v1alpha1"
		pol.Spec["kind"] = "ClientTrafficPolicy"
		pol.Spec["telemetry"] = map[string]any{
			"accessLog": map[string]any{
				"disable": !enabled,
			},
		}
	}
	ctx.Policies = append(ctx.Policies, pol)
	ctx.AddFinding(key, value, ir.StatusRequiresPolicy, adapters.Level2,
		"ClientTrafficPolicy / access logging",
		"Access log toggle mapped to provider telemetry / ClientTrafficPolicy")
	return nil
}

// CustomHTTPErrorsAdapter maps custom-http-errors → error-page backend policy scaffold.
type CustomHTTPErrorsAdapter struct{}

func (CustomHTTPErrorsAdapter) Name() string          { return "custom-http-errors" }
func (CustomHTTPErrorsAdapter) Level() adapters.Level { return adapters.Level2 }
func (CustomHTTPErrorsAdapter) CanHandle(key string) bool {
	return key == AnnCustomHTTPErrors
}

func (CustomHTTPErrorsAdapter) Transform(key, value string, ctx *adapters.Context) error {
	codes := splitCSV(value)
	pol := ir.PolicyIR{
		Kind:      ir.PolicyKind("CustomErrors"),
		Name:      ctx.Meta.IngressName + "-custom-errors",
		Namespace: ctx.Meta.Namespace,
		Provider:  ctx.Provider,
		TargetRef: ir.ParentRefIR{Name: ctx.Meta.IngressName, Namespace: ctx.Meta.Namespace},
		Spec: map[string]any{
			"apiVersion":  "gateshift.io/v1alpha1",
			"kind":        "CustomErrorPolicy",
			"statusCodes": codes,
			"note":        "Point at a custom default-backend / error pages service; no portable Core filter",
		},
	}
	if ctx.Provider == ir.ProviderEnvoyGateway {
		pol.Spec["apiVersion"] = "gateway.envoyproxy.io/v1alpha1"
		pol.Spec["kind"] = "BackendTrafficPolicy"
		pol.Spec["healthCheck"] = map[string]any{
			"note": "Use Envoy custom response / ext_proc or dedicated error page route for codes: " + strings.Join(codes, ","),
		}
	}
	ctx.Policies = append(ctx.Policies, pol)
	ctx.AddFinding(key, value, ir.StatusRequiresPolicy, adapters.Level2,
		"Custom error pages / BackendTrafficPolicy",
		fmt.Sprintf("custom-http-errors (%s) scaffolded as provider error-page policy", strings.Join(codes, ",")))
	return nil
}

// DefaultBackendAdapter maps annotation default-backend → HTTPRoute catch-all backend intent.
type DefaultBackendAdapter struct{}

func (DefaultBackendAdapter) Name() string          { return "default-backend" }
func (DefaultBackendAdapter) Level() adapters.Level { return adapters.Level1 }
func (DefaultBackendAdapter) CanHandle(key string) bool {
	return key == AnnDefaultBackend
}

func (DefaultBackendAdapter) Transform(key, value string, ctx *adapters.Context) error {
	value = strings.TrimSpace(value)
	if value == "" {
		ctx.AddFinding(key, value, ir.StatusUntranslatable, adapters.Level3, "", "default-backend annotation is empty")
		return nil
	}
	ctx.DefaultBackend = value
	ctx.AddFinding(key, value, ir.StatusDirect, adapters.Level1,
		"HTTPRoute.spec.rules catch-all backendRefs",
		"Annotation default-backend mapped to catch-all HTTPRoute backend (prefer spec.defaultBackend when both set)")
	return nil
}

// SSLPassthroughAdapter maps ssl-passthrough → Gateway TLS Passthrough listener intent.
type SSLPassthroughAdapter struct{}

func (SSLPassthroughAdapter) Name() string          { return "ssl-passthrough" }
func (SSLPassthroughAdapter) Level() adapters.Level { return adapters.Level2 }
func (SSLPassthroughAdapter) CanHandle(key string) bool {
	return key == AnnSSLPassthrough
}

func (SSLPassthroughAdapter) Transform(key, value string, ctx *adapters.Context) error {
	if !isTruthy(value) {
		ctx.AddFinding(key, value, ir.StatusDirect, adapters.Level1, "", "ssl-passthrough disabled")
		return nil
	}
	ctx.SSLPassthrough = true
	if ctx.TLS == nil {
		ctx.TLS = &ir.TLSIR{Mode: "Passthrough"}
	} else {
		ctx.TLS.Mode = "Passthrough"
	}
	ctx.AddFinding(key, value, ir.StatusRequiresPolicy, adapters.Level2,
		"Gateway.listeners[].tls.mode=Passthrough / TLSRoute",
		"ssl-passthrough requires TLS Passthrough listener (and often TLSRoute instead of HTTPRoute)")
	return nil
}

// BasicAuthAdapter maps auth-type + auth-secret → SecurityPolicy basic auth scaffold.
type BasicAuthAdapter struct{}

func (BasicAuthAdapter) Name() string          { return "basic-auth" }
func (BasicAuthAdapter) Level() adapters.Level { return adapters.Level2 }
func (BasicAuthAdapter) CanHandle(key string) bool {
	return key == AnnAuthType || key == AnnAuthSecret
}

func (BasicAuthAdapter) Transform(key, value string, ctx *adapters.Context) error {
	for _, s := range []string{AnnAuthType, AnnAuthSecret} {
		if s != key && ctx.Claimed[s] {
			ctx.Claim(key)
			return nil
		}
	}
	authType := strings.ToLower(strings.TrimSpace(ctx.Annotations[AnnAuthType]))
	secret := strings.TrimSpace(ctx.Annotations[AnnAuthSecret])
	if authType == "" {
		authType = "basic"
	}
	pol := ir.PolicyIR{
		Kind:      ir.PolicyKind("BasicAuth"),
		Name:      ctx.Meta.IngressName + "-basicauth",
		Namespace: ctx.Meta.Namespace,
		Provider:  ctx.Provider,
		TargetRef: ir.ParentRefIR{Name: ctx.Meta.IngressName, Namespace: ctx.Meta.Namespace},
		Spec: map[string]any{
			"apiVersion": "gateshift.io/v1alpha1",
			"kind":       "BasicAuthPolicy",
			"authType":   authType,
			"secretName": secret,
		},
	}
	if ctx.Provider == ir.ProviderEnvoyGateway {
		pol.Spec["apiVersion"] = "gateway.envoyproxy.io/v1alpha1"
		pol.Spec["kind"] = "SecurityPolicy"
		pol.Spec["basicAuth"] = map[string]any{
			"users": map[string]any{
				"name": secret,
			},
		}
	}
	ctx.Policies = append(ctx.Policies, pol)
	ctx.AddFinding(AnnAuthType, authType, ir.StatusRequiresPolicy, adapters.Level2,
		"SecurityPolicy.basicAuth",
		"Basic auth annotations scaffolded as SecurityPolicy — ensure htpasswd Secret exists")
	if secret != "" {
		ctx.AddFinding(AnnAuthSecret, secret, ir.StatusRequiresPolicy, adapters.Level2,
			"SecurityPolicy.basicAuth.users",
			"Consumed by basic-auth adapter")
	}
	ctx.Claim(AnnAuthType, AnnAuthSecret)
	return nil
}
