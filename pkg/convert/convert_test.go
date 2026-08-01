package convert

import (
	"strings"
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gateshift/gateshift/pkg/ir"
)

func TestFromIngressAndEmitYAML(t *testing.T) {
	pathType := networkingv1.PathTypePrefix
	class := "nginx"
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "checkout",
			Namespace: "shop",
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "/",
				"nginx.ingress.kubernetes.io/ssl-redirect":   "true",
				"nginx.ingress.kubernetes.io/limit-rps":      "50",
				"cert-manager.io/cluster-issuer":             "letsencrypt",
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &class,
			TLS: []networkingv1.IngressTLS{{
				Hosts:      []string{"checkout.example.com"},
				SecretName: "checkout-tls",
			}},
			Rules: []networkingv1.IngressRule{{
				Host: "checkout.example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/api",
							PathType: &pathType,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: "checkout-svc",
									Port: networkingv1.ServiceBackendPort{Number: 8080},
								},
							},
						}},
					},
				},
			}},
		},
	}

	bundle, err := FromIngress(ing, Options{
		Provider:       ir.ProviderEnvoyGateway,
		GatewayClass:   "envoy",
		IncludeGateway: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.HTTPRoutes) != 1 {
		t.Fatalf("expected 1 HTTPRoute, got %d", len(bundle.HTTPRoutes))
	}
	if len(bundle.Gateways) != 1 {
		t.Fatalf("expected 1 Gateway, got %d", len(bundle.Gateways))
	}
	if len(bundle.Policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(bundle.Policies))
	}

	out, err := EmitYAML(bundle)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, want := range []string{
		"kind: Gateway",
		"kind: HTTPRoute",
		"checkout.example.com",
		"checkout-svc",
		"BackendTrafficPolicy",
		"kind: Certificate",
		"letsencrypt",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("output missing %q\n%s", want, text)
		}
	}
}
