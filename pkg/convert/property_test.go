package convert_test

import (
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gateshift/gateshift/pkg/convert"
	"github.com/gateshift/gateshift/pkg/ir"
)

func sampleIngress(name string, anns map[string]string) *networkingv1.Ingress {
	pt := networkingv1.PathTypePrefix
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "demo", Annotations: anns},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				Host: name + ".example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/",
							PathType: &pt,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: "svc", Port: networkingv1.ServiceBackendPort{Number: 80},
								},
							},
						}},
					},
				},
			}},
		},
	}
}

func TestIRInvariants(t *testing.T) {
	bundle, err := convert.FromIngresses([]*networkingv1.Ingress{
		sampleIngress("a", map[string]string{
			"nginx.ingress.kubernetes.io/rewrite-target": "/x",
			"nginx.ingress.kubernetes.io/limit-rps":      "10",
		}),
	}, convert.Options{Provider: ir.ProviderEnvoyGateway, IncludeGateway: true, GatewayClass: "envoy"})
	if err != nil {
		t.Fatal(err)
	}
	if bundle.SchemaVersion != ir.SchemaVersion {
		t.Fatalf("missing schema version: %q", bundle.SchemaVersion)
	}
	if len(bundle.RequiredFeatures) == 0 {
		t.Fatal("expected RequiredFeatures")
	}
	hasHTTP := false
	for _, f := range bundle.RequiredFeatures {
		if f == ir.FeatureHTTPRoute {
			hasHTTP = true
		}
	}
	if !hasHTTP {
		t.Fatal("HTTPRoute must be required")
	}
	for _, f := range bundle.Findings {
		if f.ID == "" {
			t.Fatalf("finding missing ID: %#v", f)
		}
		if f.Severity == "" {
			t.Fatalf("finding missing severity: %#v", f)
		}
	}
	// Determinism: two converts produce identical IR JSON.
	b2, err := convert.FromIngresses([]*networkingv1.Ingress{
		sampleIngress("a", map[string]string{
			"nginx.ingress.kubernetes.io/rewrite-target": "/x",
			"nginx.ingress.kubernetes.io/limit-rps":      "10",
		}),
	}, convert.Options{Provider: ir.ProviderEnvoyGateway, IncludeGateway: true, GatewayClass: "envoy"})
	if err != nil {
		t.Fatal(err)
	}
	j1, _ := convert.MarshalIRJSON(bundle)
	j2, _ := convert.MarshalIRJSON(b2)
	if string(j1) != string(j2) {
		t.Fatal("IR JSON not deterministic across identical converts")
	}
}

func TestEveryRouteRuleHasMatchOrRedirect(t *testing.T) {
	bundle, err := convert.FromIngresses([]*networkingv1.Ingress{
		sampleIngress("r", map[string]string{
			"nginx.ingress.kubernetes.io/ssl-redirect": "true",
		}),
	}, convert.Options{Provider: ir.ProviderStandard, IncludeGateway: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range bundle.HTTPRoutes {
		if len(route.Rules) == 0 {
			t.Fatal("HTTPRoute with zero rules")
		}
		for _, rule := range route.Rules {
			hasRedirect := false
			for _, f := range rule.Filters {
				if f.Kind == ir.FilterRequestRedirect {
					hasRedirect = true
				}
			}
			if len(rule.Matches) == 0 && !hasRedirect {
				t.Fatalf("rule has neither matches nor redirect: %#v", rule)
			}
			if hasRedirect && len(rule.Backends) > 0 {
				t.Fatal("RequestRedirect rule must not include backends")
			}
		}
	}
}

func FuzzFromIngresses(f *testing.F) {
	f.Add("rewrite", "/app", true)
	f.Add("limit", "50", false)
	f.Add("unknown", "1", true)
	f.Fuzz(func(t *testing.T, keySuffix, value string, includeGW bool) {
		if len(keySuffix) > 64 || len(value) > 256 {
			return
		}
		anns := map[string]string{}
		if keySuffix != "" {
			anns["nginx.ingress.kubernetes.io/"+keySuffix] = value
		}
		ing := sampleIngress("fuzz", anns)
		bundle, err := convert.FromIngresses([]*networkingv1.Ingress{ing}, convert.Options{
			Provider:       ir.ProviderEnvoyGateway,
			IncludeGateway: includeGW,
			GatewayClass:   "envoy",
		})
		if err != nil {
			t.Fatal(err)
		}
		if bundle.SchemaVersion != ir.SchemaVersion {
			t.Fatal("schema missing after fuzz convert")
		}
		for _, finding := range bundle.Findings {
			if finding.ID == "" {
				t.Fatal("empty finding id")
			}
		}
		// Multi-doc invariant: gateway optional, routes always present for rules Ingress.
		if len(bundle.HTTPRoutes) == 0 {
			t.Fatal("expected HTTPRoute")
		}
	})
}
