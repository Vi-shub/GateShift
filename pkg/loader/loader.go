// Package loader reads Ingress manifests from YAML files.
package loader

import (
	"bytes"
	"fmt"
	"io"
	"os"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

// LoadIngressFile parses one or more Ingress resources from a YAML file.
func LoadIngressFile(path string) ([]*networkingv1.Ingress, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LoadIngressBytes(data)
}

// LoadIngressBytes parses Ingress resources from raw YAML bytes.
func LoadIngressBytes(data []byte) ([]*networkingv1.Ingress, error) {
	dec := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
	var out []*networkingv1.Ingress
	for {
		u := &unstructured.Unstructured{}
		if err := dec.Decode(u); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode yaml: %w", err)
		}
		if len(u.Object) == 0 {
			continue
		}
		if u.GetKind() != "Ingress" {
			continue
		}
		ing, err := unstructuredToIngress(u)
		if err != nil {
			return nil, err
		}
		out = append(out, ing)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no Ingress resources found in input")
	}
	return out, nil
}

func unstructuredToIngress(u *unstructured.Unstructured) (*networkingv1.Ingress, error) {
	ing := &networkingv1.Ingress{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, ing); err != nil {
		return nil, fmt.Errorf("convert Ingress %s/%s: %w", u.GetNamespace(), u.GetName(), err)
	}
	if ing.Namespace == "" {
		ing.Namespace = "default"
	}
	ing.TypeMeta = metav1.TypeMeta{
		APIVersion: "networking.k8s.io/v1",
		Kind:       "Ingress",
	}
	return ing, nil
}
