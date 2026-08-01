package nginx

import (
	"testing"

	"github.com/gateshift/gateshift/pkg/adapters"
	"github.com/gateshift/gateshift/pkg/ir"
)

func TestTranslateRewriteAndSSLRedirect(t *testing.T) {
	ann := map[string]string{
		AnnRewriteTarget: "/app",
		AnnSSLRedirect:   "true",
	}
	res := Translate(ann, ir.ProviderStandard, AuditMeta{IngressName: "demo", Namespace: "default"})
	if len(res.Filters) != 2 {
		t.Fatalf("expected 2 filters, got %d", len(res.Filters))
	}
	if res.Filters[0].Kind != ir.FilterURLRewrite && res.Filters[1].Kind != ir.FilterURLRewrite {
		t.Fatalf("expected URLRewrite among filters: %#v", res.Filters)
	}
}

func TestTranslateLimitRPSRequiresPolicy(t *testing.T) {
	ann := map[string]string{AnnLimitRPS: "100"}
	res := Translate(ann, ir.ProviderEnvoyGateway, AuditMeta{IngressName: "api", Namespace: "prod"})
	if len(res.Policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(res.Policies))
	}
	if res.Policies[0].Spec["kind"] != "BackendTrafficPolicy" {
		t.Fatalf("expected BackendTrafficPolicy, got %#v", res.Policies[0].Spec["kind"])
	}
	rl, ok := res.Policies[0].Spec["rateLimit"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested rateLimit for envoy-gateway, got %#v", res.Policies[0].Spec)
	}
	if rl["local"] == nil {
		t.Fatalf("expected local rateLimit rules, got %#v", rl)
	}
}

func TestTranslateSnippetUntranslatable(t *testing.T) {
	ann := map[string]string{
		AnnConfigurationSnippet: `more_set_headers "X-Custom: True"; if ($http_user_agent ~* "bad-bot") { return 403; }`,
	}
	res := Translate(ann, ir.ProviderStandard, AuditMeta{IngressName: "x", Namespace: "ns"})
	if len(res.Findings) == 0 || res.Findings[0].Status != ir.StatusUntranslatable {
		t.Fatalf("expected untranslatable finding, got %#v", res.Findings)
	}
	if res.Findings[0].Level != int(adapters.Level3) {
		t.Fatalf("expected L3, got %d", res.Findings[0].Level)
	}
	if res.Findings[0].Message == "" || !contains(res.Findings[0].Message, "hints:") {
		t.Fatalf("expected actionable hints, got %q", res.Findings[0].Message)
	}
}

func TestTranslateCORS(t *testing.T) {
	ann := map[string]string{
		AnnCORSEnable:      "true",
		AnnCORSAllowOrigin: "https://example.com",
	}
	res := Translate(ann, ir.ProviderStandard, AuditMeta{IngressName: "web", Namespace: "default"})
	if len(res.Filters) != 1 || res.Filters[0].Kind != ir.FilterResponseHeader {
		t.Fatalf("expected response header filter, got %#v", res.Filters)
	}
}

func TestTranslateCertManager(t *testing.T) {
	ann := map[string]string{AnnCertManagerClusterIssuer: "letsencrypt"}
	res := Translate(ann, ir.ProviderStandard, AuditMeta{IngressName: "shop", Namespace: "default"})
	if res.TLS == nil || res.TLS.ClusterIssuer != "letsencrypt" {
		t.Fatalf("expected TLS cluster issuer, got %#v", res.TLS)
	}
}

func TestPluginRegistryCoversKeys(t *testing.T) {
	reg := NewRegistry()
	keys := []string{AnnRewriteTarget, AnnLimitRPS, AnnConfigurationSnippet, AnnWhitelistSourceRange}
	for _, k := range keys {
		handled := false
		for _, a := range DefaultAdapters() {
			if a.CanHandle(k) {
				handled = true
				break
			}
		}
		if !handled {
			t.Fatalf("no adapter for %s (registry=%v)", k, reg)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
