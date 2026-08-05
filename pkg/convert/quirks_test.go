package convert_test

import (
	"strings"
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gateshift/gateshift/pkg/convert"
	"github.com/gateshift/gateshift/pkg/ir"
)

func TestFromIngressesEmitsQuirkFindings(t *testing.T) {
	pt := networkingv1.PathTypeExact
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "slash",
			Namespace: "demo",
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				Host: "trailing-slash.example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/my-path/",
							PathType: &pt,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: "httpbin",
									Port: networkingv1.ServiceBackendPort{Number: 8000},
								},
							},
						}},
					},
				},
			}},
		},
	}
	bundle, err := convert.FromIngresses([]*networkingv1.Ingress{ing}, convert.Options{
		Provider: ir.ProviderEnvoyGateway, IncludeGateway: true, GatewayClass: "envoy",
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range bundle.Findings {
		if f.Key == "gateshift.io/nginx-quirk/trailing-slash" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected trailing-slash quirk in audit findings")
	}
}

func TestPreserveNGINXRegexAndTrailingSlash(t *testing.T) {
	pt := networkingv1.PathTypeExact
	impl := networkingv1.PathTypeImplementationSpecific
	a := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "regex",
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/use-regex": "true",
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				Host: "regex-match.example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/[A-Z]{3}",
							PathType: &impl,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: "httpbin", Port: networkingv1.ServiceBackendPort{Number: 8000},
								},
							},
						}},
					},
				},
			}},
		},
	}
	b := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "slash"},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				Host: "trailing-slash.example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/my-path/",
							PathType: &pt,
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: "httpbin", Port: networkingv1.ServiceBackendPort{Number: 8000},
								},
							},
						}},
					},
				},
			}},
		},
	}

	bundle, err := convert.FromIngresses([]*networkingv1.Ingress{a, b}, convert.Options{
		Provider:                   ir.ProviderEnvoyGateway,
		IncludeGateway:             true,
		GatewayClass:               "envoy",
		PreserveNGINXRegex:         true,
		EmitTrailingSlashRedirects: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var preserved, emittedSlash bool
	for _, f := range bundle.Findings {
		if f.Key == "gateshift.io/nginx-quirk/preserve-regex" {
			preserved = true
			if !strings.Contains(f.Value, "(?i)") {
				t.Fatalf("preserve value should be case-insensitive regex: %s", f.Value)
			}
		}
		if f.Key == "gateshift.io/nginx-quirk/trailing-slash-emit" {
			emittedSlash = true
		}
	}
	if !preserved {
		t.Fatal("expected preserve-regex finding")
	}
	if !emittedSlash {
		t.Fatal("expected trailing-slash-emit finding")
	}
}
