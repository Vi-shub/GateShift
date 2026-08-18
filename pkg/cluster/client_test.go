package cluster

import (
	"context"
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListIngressesNamespaceAndSelector(t *testing.T) {
	cs := fake.NewSimpleClientset(
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "shop",
				Namespace: "prod",
				Labels:    map[string]string{"team": "checkout", "env": "prod"},
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "blog",
				Namespace: "prod",
				Labels:    map[string]string{"team": "content", "env": "prod"},
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "staging-shop",
				Namespace: "staging",
				Labels:    map[string]string{"team": "checkout", "env": "staging"},
			},
		},
	)
	cl := NewFromClientset(cs)
	ctx := context.Background()

	t.Run("namespace only", func(t *testing.T) {
		got, err := cl.ListIngresses(ctx, ListOptions{Namespace: "prod"})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("want 2, got %d", len(got))
		}
	})

	t.Run("label selector", func(t *testing.T) {
		got, err := cl.ListIngresses(ctx, ListOptions{
			Namespace:     "prod",
			LabelSelector: "team=checkout",
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Name != "shop" {
			t.Fatalf("want shop, got %#v", got)
		}
	})

	t.Run("all namespaces with selector", func(t *testing.T) {
		got, err := cl.ListIngresses(ctx, ListOptions{
			LabelSelector: "team=checkout",
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("want 2 checkout ingresses, got %d", len(got))
		}
	})
}
