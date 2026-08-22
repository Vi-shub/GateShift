package nginx

import (
	"testing"

	"github.com/gateshift/gateshift/pkg/ir"
)

func TestAffinityConsumesCookieFamily(t *testing.T) {
	ann := map[string]string{
		AnnAffinity:                             "cookie",
		AnnSessionCookieName:                    "route",
		AnnSessionCookieExpires:                 "172800",
		AnnSessionCookieMaxAge:                  "172800",
		AnnSessionCookieSecure:                  "true",
		AnnSessionCookieSameSite:                "None",
		AnnSessionCookieConditionalSameSiteNone: "true",
	}
	res := Translate(ann, ir.ProviderEnvoyGateway, AuditMeta{IngressName: "sticky", Namespace: "default"})
	if len(res.Policies) != 1 {
		t.Fatalf("expected 1 affinity policy, got %d", len(res.Policies))
	}
	sp, ok := res.Policies[0].Spec["loadBalancer"].(map[string]any)
	if !ok {
		t.Fatalf("expected loadBalancer for envoy-gateway, got %#v", res.Policies[0].Spec)
	}
	ch, ok := sp["consistentHash"].(map[string]any)
	if !ok || ch["type"] != "Cookie" {
		t.Fatalf("expected ConsistentHash Cookie, got %#v", sp)
	}
	cookie, ok := ch["cookie"].(map[string]any)
	if !ok || cookie["name"] != "route" || cookie["ttl"] != "172800s" {
		t.Fatalf("expected cookie name/ttl, got %#v", cookie)
	}
	// Must not leak IR-only fields into BackendTrafficPolicy.
	for _, bad := range []string{"affinity", "cookieName", "featureGate", "sessionPersistence", "cookieMaxAge"} {
		if _, exists := res.Policies[0].Spec[bad]; exists {
			t.Fatalf("unexpected IR field %q in EG policy spec: %#v", bad, res.Policies[0].Spec)
		}
	}
	for _, f := range res.Findings {
		if f.Status == ir.StatusUntranslatable {
			t.Fatalf("unexpected untranslatable finding after cookie gap fill: %+v", f)
		}
	}
	// All cookie keys should be claimed / reported, none left as unknown L3.
	seen := map[string]bool{}
	for _, f := range res.Findings {
		seen[f.Key] = true
	}
	for _, k := range []string{AnnSessionCookieExpires, AnnSessionCookieMaxAge, AnnSessionCookieSameSite, AnnSessionCookieSecure} {
		if !seen[k] {
			t.Fatalf("missing finding for %s", k)
		}
	}
}
