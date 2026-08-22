package convert

import (
	"strings"
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/gateshift/gateshift/pkg/ir"
)

// knownEGBackendTrafficPolicySpecKeys are fields accepted by EG v1.2 BackendTrafficPolicySpec
// (plus targetRefs / targetRef / targetSelectors). Keep in sync with EG API docs when bumping.
var knownEGBackendTrafficPolicySpecKeys = map[string]bool{
	"targetRef": true, "targetRefs": true, "targetSelectors": true,
	"loadBalancer": true, "retry": true, "proxyProtocol": true, "tcpKeepalive": true,
	"healthCheck": true, "circuitBreaker": true, "timeout": true, "connection": true,
	"dns": true, "http2": true, "rateLimit": true, "faultInjection": true,
	"useClientProtocol": true, "responseOverride": true, "mergeType": true,
}

func TestPodinfoDualRunYAMLHasOnlyKnownEGPolicyFields(t *testing.T) {
	pathType := networkingv1.PathTypePrefix
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podinfo",
			Namespace: "podinfo",
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target":         "/",
				"nginx.ingress.kubernetes.io/ssl-redirect":           "false",
				"nginx.ingress.kubernetes.io/enable-cors":            "true",
				"nginx.ingress.kubernetes.io/cors-allow-origin":      "*",
				"nginx.ingress.kubernetes.io/affinity":               "cookie",
				"nginx.ingress.kubernetes.io/session-cookie-name":    "PODINFOCOOKIE",
				"nginx.ingress.kubernetes.io/session-cookie-max-age": "86400",
				"nginx.ingress.kubernetes.io/session-cookie-secure":  "false",
				"nginx.ingress.kubernetes.io/proxy-body-size":        "8m",
				"nginx.ingress.kubernetes.io/proxy-read-timeout":     "60",
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				Host: "podinfo.local",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/",
							PathType: &pathType,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: "podinfo", Port: networkingv1.ServiceBackendPort{Number: 9898},
								},
							},
						}},
					},
				},
			}},
		},
	}

	bundle, err := FromIngress(ing, Options{
		Provider: ir.ProviderEnvoyGateway, GatewayClass: "envoy", IncludeGateway: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	ApplyDualRunMode(bundle, DualRunOptions{IncludeGateway: true})
	out, err := EmitYAML(bundle)
	if err != nil {
		t.Fatal(err)
	}

	docs := strings.Split(string(out), "\n---\n")
	var btpCount int
	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		var obj map[string]any
		if err := yaml.Unmarshal([]byte(doc), &obj); err != nil {
			t.Fatalf("yaml: %v\n%s", err, doc)
		}
		if obj["kind"] != "BackendTrafficPolicy" {
			continue
		}
		btpCount++
		spec, _ := obj["spec"].(map[string]any)
		for k := range spec {
			if !knownEGBackendTrafficPolicySpecKeys[k] {
				t.Fatalf("unknown BackendTrafficPolicy.spec.%s (would fail kubectl apply):\n%s", k, doc)
			}
		}
		refs, ok := spec["targetRefs"].([]any)
		if !ok || len(refs) == 0 {
			t.Fatalf("expected targetRefs, got %#v", spec)
		}
		ref, _ := refs[0].(map[string]any)
		if _, hasNS := ref["namespace"]; hasNS {
			t.Fatalf("targetRefs must not include namespace: %#v", ref)
		}
		if ref["name"] != "podinfo-shadow" {
			t.Fatalf("dual-run policy should target shadow route, got %#v", ref)
		}
	}
	if btpCount != 2 {
		t.Fatalf("expected 2 BackendTrafficPolicies (affinity+backend), got %d\n%s", btpCount, out)
	}
	for _, want := range []string{"kind: Gateway", "kind: HTTPRoute", "podinfo-shadow", "ConsistentHash", "requestTimeout: 60s", "bufferLimit: 8Mi"} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("missing %q", want)
		}
	}
}
