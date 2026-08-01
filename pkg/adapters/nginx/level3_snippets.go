package nginx

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gateshift/gateshift/pkg/adapters"
	"github.com/gateshift/gateshift/pkg/ir"
)

var (
	reSetHeader = regexp.MustCompile(`(?i)more_set_headers\s+"([^:]+):\s*([^"]+)"`)
	reReturn    = regexp.MustCompile(`(?i)return\s+(\d+)`)
	reIfUA      = regexp.MustCompile(`(?i)if\s*\(\s*\$http_user_agent`)
)

// SnippetAdapter detects raw nginx/Lua injections (Level 3).
// It never invents Envoy config; it produces actionable audit findings and,
// when a trivial pattern is recognized, optional hints for manual remapping.
type SnippetAdapter struct{}

func (SnippetAdapter) Name() string          { return "snippets" }
func (SnippetAdapter) Level() adapters.Level { return adapters.Level3 }
func (SnippetAdapter) CanHandle(key string) bool {
	return key == AnnConfigurationSnippet || key == AnnServerSnippet || key == AnnModsecuritySnippet
}

func (SnippetAdapter) Transform(key, value string, ctx *adapters.Context) error {
	hints := analyzeSnippet(value)
	msg := "NGINX snippets/Lua cannot be auto-translated; rewrite as Gateway filters or controller policies manually"
	if len(hints) > 0 {
		msg = msg + " | hints: " + strings.Join(hints, "; ")
	}
	ctx.AddFinding(key, truncate(value, 120), ir.StatusUntranslatable, adapters.Level3, "", msg)
	return nil
}

func analyzeSnippet(snippet string) []string {
	var hints []string
	if m := reSetHeader.FindAllStringSubmatch(snippet, -1); len(m) > 0 {
		for _, g := range m {
			hints = append(hints, fmt.Sprintf("consider ResponseHeaderModifier set %s=%s", strings.TrimSpace(g[1]), strings.TrimSpace(g[2])))
		}
	}
	if reIfUA.MatchString(snippet) {
		hints = append(hints, "user-agent deny/allow needs SecurityPolicy or Lua→Wasm rewrite")
	}
	if m := reReturn.FindStringSubmatch(snippet); len(m) == 2 {
		hints = append(hints, fmt.Sprintf("HTTP %s response may map to HTTPRoute DirectResponse (if supported) or filter chain", m[1]))
	}
	if strings.Contains(snippet, "lua_") || strings.Contains(snippet, "access_by_lua") {
		hints = append(hints, "Lua blocks require manual redesign — no portable Gateway API equivalent")
	}
	return hints
}

// AuthAdapter flags external auth annotations as Level 3 (or Level 2 hint for Envoy SecurityPolicy).
type AuthAdapter struct{}

func (AuthAdapter) Name() string          { return "auth" }
func (AuthAdapter) Level() adapters.Level { return adapters.Level3 }
func (AuthAdapter) CanHandle(key string) bool {
	return key == AnnAuthURL || key == AnnAuthSignin || key == AnnAuthTLSSecret
}

func (AuthAdapter) Transform(key, value string, ctx *adapters.Context) error {
	if ctx.Provider == ir.ProviderEnvoyGateway && key == AnnAuthURL {
		pol := ir.PolicyIR{
			Kind:      ir.PolicyKind("ExtAuth"),
			Name:      ctx.Meta.IngressName + "-extauth",
			Namespace: ctx.Meta.Namespace,
			Provider:  ctx.Provider,
			TargetRef: ir.ParentRefIR{Name: ctx.Meta.IngressName, Namespace: ctx.Meta.Namespace},
			Spec: map[string]any{
				"apiVersion": "gateway.envoyproxy.io/v1alpha1",
				"kind":       "SecurityPolicy",
				"extAuth": map[string]any{
					"http": map[string]any{
						"backendRefs": []any{},
						"uri":         value,
					},
				},
				"note": "Populate backendRefs to your auth service; GateShift only scaffolds the SecurityPolicy",
			},
		}
		ctx.Policies = append(ctx.Policies, pol)
		ctx.AddFinding(key, value, ir.StatusRequiresPolicy, adapters.Level2,
			"SecurityPolicy.extAuth",
			"auth-url scaffolded as Envoy Gateway SecurityPolicy — complete backendRefs manually")
		return nil
	}
	ctx.AddFinding(key, value, ir.StatusUntranslatable, adapters.Level3, "",
		"External auth has no portable Gateway API Core equivalent; use controller SecurityPolicy/WASM")
	return nil
}
