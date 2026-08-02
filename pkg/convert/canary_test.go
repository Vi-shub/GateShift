package convert

import (
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gateshift/gateshift/pkg/ir"
)

func TestFromIngressesMergesCanaryWeights(t *testing.T) {
	pathType := networkingv1.PathTypePrefix
	primary := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "shop", Namespace: "prod"},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				Host: "shop.example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/",
							PathType: &pathType,
							Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{
								Name: "shop-stable", Port: networkingv1.ServiceBackendPort{Number: 80},
							}},
						}},
					},
				},
			}},
		},
	}
	canary := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shop-canary",
			Namespace: "prod",
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/canary":        "true",
				"nginx.ingress.kubernetes.io/canary-weight": "20",
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				Host: "shop.example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/",
							PathType: &pathType,
							Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{
								Name: "shop-canary", Port: networkingv1.ServiceBackendPort{Number: 80},
							}},
						}},
					},
				},
			}},
		},
	}

	bundle, err := FromIngresses([]*networkingv1.Ingress{primary, canary}, Options{
		Provider: ir.ProviderEnvoyGateway, GatewayClass: "envoy", IncludeGateway: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.HTTPRoutes) != 1 {
		t.Fatalf("expected 1 merged route, got %d", len(bundle.HTTPRoutes))
	}
	foundCanary := false
	for _, rule := range bundle.HTTPRoutes[0].Rules {
		for _, be := range rule.Backends {
			if be.Name == "shop-canary" && be.Weight != nil && *be.Weight == 20 {
				foundCanary = true
			}
		}
	}
	if !foundCanary {
		t.Fatalf("expected weighted canary backend, rules=%#v", bundle.HTTPRoutes[0].Rules)
	}
}
