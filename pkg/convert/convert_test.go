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

func TestEmitEnvoyBackendTrafficPolicyShape(t *testing.T) {
	pathType := networkingv1.PathTypePrefix
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podinfo",
			Namespace: "podinfo",
			Annotations: map[string]string{
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
									Name: "podinfo",
									Port: networkingv1.ServiceBackendPort{Number: 9898},
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
	out, err := EmitYAML(bundle)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, want := range []string{
		"kind: BackendTrafficPolicy",
		"targetRefs:",
		"type: ConsistentHash",
		"requestTimeout: 60s",
		"bufferLimit: 8Mi",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
	for _, bad := range []string{
		"maxRequestBodySize",
		"readTimeout:",
		"featureGate:",
		"sessionPersistence:",
	} {
		if strings.Contains(text, bad) {
			t.Fatalf("unexpected field %q in:\n%s", bad, text)
		}
	}
	if strings.Contains(text, "targetRefs:\n  - group: gateway.networking.k8s.io\n    kind: HTTPRoute\n    name: podinfo\n    namespace:") {
		t.Fatalf("targetRefs must not include namespace:\n%s", text)
	}
}

func TestHTTPOnlySkipsTLSAndCertificates(t *testing.T) {
	pathType := networkingv1.PathTypePrefix
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "checkout",
			Namespace: "shop",
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/ssl-redirect": "true",
				"cert-manager.io/cluster-issuer":           "letsencrypt",
			},
		},
		Spec: networkingv1.IngressSpec{
			TLS: []networkingv1.IngressTLS{{
				Hosts:      []string{"checkout.example.com"},
				SecretName: "checkout-tls",
			}},
			Rules: []networkingv1.IngressRule{{
				Host: "checkout.example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/",
							PathType: &pathType,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: "checkout-svc",
									Port: networkingv1.ServiceBackendPort{Number: 80},
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
		HTTPOnly:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Certificates) != 0 {
		t.Fatalf("expected no certificates, got %d", len(bundle.Certificates))
	}
	if len(bundle.Gateways) != 1 {
		t.Fatalf("expected 1 gateway, got %d", len(bundle.Gateways))
	}
	listeners := bundle.Gateways[0].Listeners
	if len(listeners) != 1 || listeners[0].Protocol != "HTTP" {
		t.Fatalf("expected single HTTP listener, got %#v", listeners)
	}
	foundHTTPOnly := false
	for _, f := range bundle.Findings {
		if f.ID == ir.FindingIDHTTPOnly {
			foundHTTPOnly = true
			break
		}
	}
	if !foundHTTPOnly {
		t.Fatal("expected convert.http-only finding")
	}

	out, err := EmitYAML(bundle)
	if err != nil {
		t.Fatal(err)
	}
	text := string(out)
	for _, bad := range []string{"kind: Certificate", "protocol: HTTPS", "checkout-tls", "scheme: https"} {
		if strings.Contains(text, bad) {
			t.Fatalf("http-only output unexpectedly contains %q\n%s", bad, text)
		}
	}
}
