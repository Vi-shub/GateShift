package nginx

import (
	"testing"

	"github.com/gateshift/gateshift/pkg/ir"
)

func TestCatalogFullyImplemented(t *testing.T) {
	stats := CatalogCoverage()
	if len(stats.MissingKeys) != 0 {
		t.Fatalf("catalog still has gaps: %v", stats.MissingKeys)
	}
	if stats.Percent < 100 {
		t.Fatalf("expected 100%% catalog coverage, got %.1f%% (%d/%d)", stats.Percent, stats.Implemented, stats.Total)
	}
}

func TestGapAdaptersTranslate(t *testing.T) {
	ann := map[string]string{
		AnnEnableAccessLog:      "true",
		AnnCustomHTTPErrors:     "404,503",
		AnnDefaultBackend:       "default/fallback-svc",
		AnnProxyBuffering:       "off",
		AnnProxyNextUpstream:    "error timeout",
		AnnSSLPassthrough:       "true",
		AnnAuthType:             "basic",
		AnnAuthSecret:           "basic-auth",
		AnnClientBodyBufferSize: "16k",
		AnnProxyConnectTimeout:  "5",
	}
	res := Translate(ann, ir.ProviderEnvoyGateway, AuditMeta{IngressName: "gap", Namespace: "demo"})
	if res.DefaultBackend != "default/fallback-svc" {
		t.Fatalf("default backend not set: %q", res.DefaultBackend)
	}
	if !res.SSLPassthrough {
		t.Fatal("expected SSLPassthrough")
	}
	for _, f := range res.Findings {
		if f.Status == ir.StatusUntranslatable {
			t.Fatalf("unexpected L3 for catalog gap key %s: %s", f.Key, f.Message)
		}
	}
	// EG path: backend tuning + basicAuth policies. access-log / custom-http-errors stay findings-only
	// (no invalid ClientTrafficPolicy / BackendTrafficPolicy shapes).
	if len(res.Policies) < 2 {
		t.Fatalf("expected backend + basicAuth policies from gap adapters, got %d", len(res.Policies))
	}
	kinds := map[string]bool{}
	for _, p := range res.Policies {
		if k, ok := p.Spec["kind"].(string); ok {
			kinds[k] = true
		}
	}
	if !kinds["BackendTrafficPolicy"] || !kinds["SecurityPolicy"] {
		t.Fatalf("expected BackendTrafficPolicy + SecurityPolicy, got %#v", kinds)
	}
	for _, p := range res.Policies {
		for _, bad := range []string{"enabled", "note", "statusCodes", "authType", "secretName"} {
			if _, ok := p.Spec[bad]; ok {
				t.Fatalf("IR field %q leaked into EG policy %#v", bad, p.Spec)
			}
		}
	}
}
