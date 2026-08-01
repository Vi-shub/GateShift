package loader

import (
	"testing"
)

func TestLoadIngressBytes(t *testing.T) {
	yaml := []byte(`
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: demo
  namespace: app
spec:
  rules:
    - host: demo.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: demo
                port:
                  number: 80
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: ignore-me
data:
  x: y
`)
	ings, err := LoadIngressBytes(yaml)
	if err != nil {
		t.Fatal(err)
	}
	if len(ings) != 1 {
		t.Fatalf("expected 1 ingress, got %d", len(ings))
	}
	if ings[0].Name != "demo" || ings[0].Namespace != "app" {
		t.Fatalf("unexpected ingress identity: %s/%s", ings[0].Namespace, ings[0].Name)
	}
}
