package convert

import (
	"testing"
)

func TestSanitizeEnvoyBackendTrafficPolicyDropsUnknown(t *testing.T) {
	spec := map[string]any{
		"targetRefs": []any{map[string]any{
			"group": "gateway.networking.k8s.io", "kind": "HTTPRoute", "name": "r",
		}},
		"timeout": map[string]any{"http": map[string]any{"requestTimeout": "60s"}},
		// Invented / newer-only / IR leftovers — must be stripped.
		"maxRequestBodySize": "8m",
		"readTimeout":        "60s",
		"affinity":           "cookie",
		"featureGate":        "SessionPersistence",
		"sessionPersistence": map[string]any{"type": "Cookie"},
		"requestBuffer":      map[string]any{"limit": "8Mi"}, // newer-only; omit for EG 1.2+ portability
		"namespace":          "podinfo",
	}
	out, ok := sanitizeEnvoyGatewayPolicySpec("BackendTrafficPolicy", spec)
	if !ok {
		t.Fatal("expected useful policy")
	}
	for _, bad := range []string{"maxRequestBodySize", "readTimeout", "affinity", "featureGate", "sessionPersistence", "requestBuffer", "namespace"} {
		if _, exists := out[bad]; exists {
			t.Fatalf("expected %q stripped, got %#v", bad, out)
		}
	}
	if out["timeout"] == nil || out["targetRefs"] == nil {
		t.Fatalf("expected timeout+targetRefs kept: %#v", out)
	}
}

func TestSanitizeEnvoySecurityPolicyDropsExtAuthWithoutBackend(t *testing.T) {
	spec := map[string]any{
		"targetRefs": []any{map[string]any{"group": "gateway.networking.k8s.io", "kind": "HTTPRoute", "name": "r"}},
		"extAuth": map[string]any{
			"http": map[string]any{
				"backendRefs": []any{},
				"uri":         "http://auth.example/verify",
			},
		},
		"note": "scaffold",
	}
	_, ok := sanitizeEnvoyGatewayPolicySpec("SecurityPolicy", spec)
	if ok {
		t.Fatal("empty extAuth scaffold should not be considered useful")
	}
}

func TestSanitizeEnvoyDropsClientTrafficPolicy(t *testing.T) {
	_, ok := sanitizeEnvoyGatewayPolicySpec("ClientTrafficPolicy", map[string]any{
		"targetRefs": []any{},
		"telemetry":  map[string]any{},
	})
	if ok {
		t.Fatal("ClientTrafficPolicy must not be emitted yet")
	}
}
