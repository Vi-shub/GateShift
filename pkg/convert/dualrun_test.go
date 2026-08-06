package convert_test

import (
	"strings"
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gateshift/gateshift/pkg/convert"
	"github.com/gateshift/gateshift/pkg/ir"
)

func TestApplyDualRunMode(t *testing.T) {
	pt := networkingv1.PathTypePrefix
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web",
			Namespace: "demo",
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "/app",
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				Host: "web.example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/",
							PathType: &pt,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: "web",
									Port: networkingv1.ServiceBackendPort{Number: 80},
								},
							},
						}},
					},
				},
			}},
		},
	}

	bundle, err := convert.FromIngresses([]*networkingv1.Ingress{ing}, convert.Options{
		Provider:       ir.ProviderEnvoyGateway,
		IncludeGateway: true,
		GatewayClass:   "envoy",
		GatewayName:    "web-staging-gateway",
	})
	if err != nil {
		t.Fatal(err)
	}
	convert.ApplyDualRunMode(bundle, convert.DualRunOptions{
		GatewayName:    "web-staging-gateway",
		IncludeGateway: true,
	})

	if len(bundle.Gateways) != 1 || bundle.Gateways[0].Name != "web-staging-gateway" {
		t.Fatalf("staging gateway: %#v", bundle.Gateways)
	}
	if bundle.Gateways[0].Annotations[convert.AnnMode] != convert.ModeDualRun {
		t.Fatalf("gateway annotations: %#v", bundle.Gateways[0].Annotations)
	}
	if len(bundle.HTTPRoutes) != 1 || bundle.HTTPRoutes[0].Name != "web-shadow" {
		t.Fatalf("shadow route: %#v", bundle.HTTPRoutes)
	}
	r := bundle.HTTPRoutes[0]
	if r.Annotations[convert.AnnShadow] != "true" || r.Annotations[convert.AnnSourceIngress] != "web" {
		t.Fatalf("route annotations: %#v", r.Annotations)
	}
	if len(r.ParentRefs) == 0 || r.ParentRefs[0].Name != "web-staging-gateway" {
		t.Fatalf("parentRefs: %#v", r.ParentRefs)
	}

	yamlBytes, err := convert.EmitYAML(bundle)
	if err != nil {
		t.Fatal(err)
	}
	s := string(yamlBytes)
	if strings.Contains(s, "kind: Ingress") {
		t.Fatal("dual-run must not emit Ingress")
	}
	if !strings.Contains(s, "gateshift.io/mode: dual-run") {
		t.Fatal("expected dual-run annotation in YAML")
	}
	if !strings.Contains(s, "name: web-shadow") {
		t.Fatal("expected shadow route name in YAML")
	}

	checklist := convert.FormatDualRunChecklist(bundle)
	if !strings.Contains(checklist, "Ingress is NOT modified") {
		t.Fatalf("checklist missing safety note: %s", checklist)
	}
}

func TestApplyDualRunNoGateway(t *testing.T) {
	pt := networkingv1.PathTypePrefix
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "demo"},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				Host: "api.example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/",
							PathType: &pt,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: "api", Port: networkingv1.ServiceBackendPort{Number: 80},
								},
							},
						}},
					},
				},
			}},
		},
	}
	bundle, err := convert.FromIngresses([]*networkingv1.Ingress{ing}, convert.Options{
		Provider: ir.ProviderStandard, IncludeGateway: false, GatewayName: "existing-gw",
	})
	if err != nil {
		t.Fatal(err)
	}
	convert.ApplyDualRunMode(bundle, convert.DualRunOptions{
		GatewayName: "existing-gw", IncludeGateway: false,
	})
	if len(bundle.Gateways) != 0 {
		t.Fatalf("expected no gateway docs, got %#v", bundle.Gateways)
	}
	if bundle.HTTPRoutes[0].ParentRefs[0].Name != "existing-gw" {
		t.Fatalf("parentRef: %#v", bundle.HTTPRoutes[0].ParentRefs)
	}
}
