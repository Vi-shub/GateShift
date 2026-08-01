package nginx

import (
	"strconv"
	"strings"

	"github.com/gateshift/gateshift/pkg/adapters"
	"github.com/gateshift/gateshift/pkg/ir"
)

// RateLimitAdapter maps limit-rps/rpm/connections → provider Policy CRDs (Level 2).
type RateLimitAdapter struct{}

func (RateLimitAdapter) Name() string          { return "rate-limit" }
func (RateLimitAdapter) Level() adapters.Level { return adapters.Level2 }
func (RateLimitAdapter) CanHandle(key string) bool {
	return key == AnnLimitRPS || key == AnnLimitRPM || key == AnnLimitConnections
}

func (RateLimitAdapter) Transform(key, value string, ctx *adapters.Context) error {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n <= 0 {
		ctx.AddFinding(key, value, ir.StatusUntranslatable, adapters.Level3, "",
			"Rate limit value is not a positive integer")
		return nil
	}

	unit := "Second"
	requests := n
	switch key {
	case AnnLimitRPM:
		unit = "Minute"
	case AnnLimitConnections:
		unit = "Second"
	}

	name := ctx.Meta.IngressName + "-ratelimit"
	pol := ir.PolicyIR{
		Kind:      ir.PolicyRateLimit,
		Name:      name,
		Namespace: ctx.Meta.Namespace,
		Provider:  ctx.Provider,
		TargetRef: ir.ParentRefIR{Name: ctx.Meta.IngressName, Namespace: ctx.Meta.Namespace},
		Spec:      map[string]any{},
	}

	switch ctx.Provider {
	case ir.ProviderEnvoyGateway:
		pol.Spec["apiVersion"] = "gateway.envoyproxy.io/v1alpha1"
		pol.Spec["kind"] = "BackendTrafficPolicy"
		pol.Spec["rateLimit"] = map[string]any{
			"local": map[string]any{
				"rules": []any{
					map[string]any{
						"clientSelectors": []any{map[string]any{}},
						"limit": map[string]any{
							"requests": requests,
							"unit":     unit,
						},
					},
				},
			},
		}
	case ir.ProviderCilium:
		pol.Spec["apiVersion"] = "cilium.io/v2"
		pol.Spec["kind"] = "CiliumEnvoyConfig"
		pol.Spec["requestsPerSecond"] = requests
		pol.Spec["unit"] = unit
	case ir.ProviderIstio:
		pol.Spec["apiVersion"] = "security.istio.io/v1"
		pol.Spec["kind"] = "AuthorizationPolicy"
		pol.Spec["requestsPerSecond"] = requests
	case ir.ProviderKong:
		pol.Spec["apiVersion"] = "configuration.konghq.com/v1"
		pol.Spec["kind"] = "KongPlugin"
		pol.Spec["plugin"] = "rate-limiting"
		pol.Spec["config"] = map[string]any{"second": requests}
	default:
		pol.Spec["apiVersion"] = "gateshift.io/v1alpha1"
		pol.Spec["kind"] = "RateLimitPolicy"
		pol.Spec["requests"] = requests
		pol.Spec["unit"] = unit
	}

	ctx.Policies = append(ctx.Policies, pol)
	ctx.AddFinding(key, value, ir.StatusRequiresPolicy, adapters.Level2,
		string(pol.Kind)+"/"+string(ctx.Provider),
		"Rate limiting requires a vendor Policy CRD extension")
	return nil
}
