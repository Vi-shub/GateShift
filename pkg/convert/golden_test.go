package convert_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gateshift/gateshift/pkg/convert"
	"github.com/gateshift/gateshift/pkg/ir"
)

var updateGoldens = flag.Bool("update-goldens", false, "rewrite IR golden files")

func TestIRGoldenBasicRewrite(t *testing.T) {
	pt := networkingv1.PathTypePrefix
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web",
			Namespace: "demo",
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "/app",
				"nginx.ingress.kubernetes.io/ssl-redirect":   "true",
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
		Provider: ir.ProviderEnvoyGateway, IncludeGateway: true, GatewayClass: "envoy",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertIRGolden(t, "basic-rewrite.json", bundle)
}

func TestIRGoldenUnknownAnnotation(t *testing.T) {
	pt := networkingv1.PathTypePrefix
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unk",
			Namespace: "demo",
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/totally-unknown-future-key": "1",
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				Host: "unk.example.com",
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
		Provider: ir.ProviderStandard, IncludeGateway: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range bundle.Findings {
		if f.ID == ir.FindingIDAnnotationUnknown {
			found = true
			if f.Status != ir.StatusRequiresPolicy {
				t.Fatalf("unknown should be requires_policy, got %s", f.Status)
			}
		}
	}
	if !found {
		t.Fatal("expected annotation.unknown finding")
	}
	if bundle.SchemaVersion != ir.SchemaVersion {
		t.Fatalf("schema: %s", bundle.SchemaVersion)
	}
	assertIRGolden(t, "unknown-annotation.json", bundle)
}

func assertIRGolden(t *testing.T, name string, bundle *ir.MigrationBundle) {
	t.Helper()
	got, err := convert.MarshalIRJSON(bundle)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join("testdata", "golden")
	path := filepath.Join(dir, name)
	if *updateGoldens {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s (run with -update-goldens): %v", path, err)
	}
	if !bytes.Equal(bytes.TrimSpace(want), bytes.TrimSpace(got)) {
		// Re-parse to give a clearer structural hint.
		var a, b any
		_ = json.Unmarshal(want, &a)
		_ = json.Unmarshal(got, &b)
		t.Fatalf("IR golden mismatch for %s\n--- want ---\n%s\n--- got ---\n%s", name, want, got)
	}
}
