package nginxquirks_test

import (
	"strings"
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gateshift/gateshift/pkg/nginxquirks"
)

func pathType(t networkingv1.PathType) *networkingv1.PathType { return &t }

func backend(name string) networkingv1.IngressBackend {
	return networkingv1.IngressBackend{
		Service: &networkingv1.IngressServiceBackend{
			Name: name,
			Port: networkingv1.ServiceBackendPort{Number: 8000},
		},
	}
}

func TestAnalyzeUseRegexHostWide(t *testing.T) {
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
							PathType: pathType(networkingv1.PathTypeImplementationSpecific),
							Backend:  backend("httpbin"),
						}},
					},
				},
			}},
		},
	}
	b := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "other"},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				Host: "regex-match.example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/Header",
							PathType: pathType(networkingv1.PathTypeExact),
							Backend:  backend("httpbin"),
						}},
					},
				},
			}},
		},
	}

	res := nginxquirks.Analyze([]*networkingv1.Ingress{a, b})
	if !res.HostForcesRegex("regex-match.example.com") {
		t.Fatal("expected host regex force")
	}
	var host, pathAsRegex bool
	for _, f := range res.Findings {
		switch f.Key {
		case "gateshift.io/nginx-quirk/host-regex":
			host = true
		case "gateshift.io/nginx-quirk/path-as-regex":
			pathAsRegex = true
		}
	}
	if !host || !pathAsRegex {
		t.Fatalf("missing findings host=%v pathAsRegex=%v findings=%d", host, pathAsRegex, len(res.Findings))
	}
}

func TestAnalyzeRewriteImpliesRegex(t *testing.T) {
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "rw",
			Annotations: map[string]string{"nginx.ingress.kubernetes.io/rewrite-target": "/uuid"},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				Host: "rewrite-target.example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/IP",
							PathType: pathType(networkingv1.PathTypeExact),
							Backend:  backend("httpbin"),
						}},
					},
				},
			}},
		},
	}
	res := nginxquirks.Analyze([]*networkingv1.Ingress{ing})
	if !res.HostForcesRegex("rewrite-target.example.com") {
		t.Fatal("rewrite-target should imply regex")
	}
}

func TestAnalyzeTrailingSlash(t *testing.T) {
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "slash"},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				Host: "trailing-slash.example.com",
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/my-path/",
							PathType: pathType(networkingv1.PathTypeExact),
							Backend:  backend("b"),
						}},
					},
				},
			}},
		},
	}
	res := nginxquirks.Analyze([]*networkingv1.Ingress{ing})
	found := false
	for _, f := range res.Findings {
		if f.Key == "gateshift.io/nginx-quirk/trailing-slash" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected trailing-slash finding")
	}
}

func TestPreserveRegexPath(t *testing.T) {
	got := nginxquirks.PreserveRegexPath("/Header")
	if !strings.HasPrefix(got, "(?i)") || !strings.HasSuffix(got, ".*") {
		t.Fatalf("unexpected preserve path: %s", got)
	}
	got2 := nginxquirks.PreserveRegexPath("/[A-Z]{3}")
	if !strings.Contains(got2, "(?i)") {
		t.Fatalf("regex body should be case-insensitive: %s", got2)
	}
}
