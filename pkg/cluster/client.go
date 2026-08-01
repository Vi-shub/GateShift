// Package cluster loads live Ingress resources from a Kubernetes cluster.
package cluster

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Options configures kube client loading.
type Options struct {
	Kubeconfig string
	Context    string
}

// Client wraps a typed kubernetes clientset.
type Client struct {
	cs kubernetes.Interface
}

// New builds a client from kubeconfig or in-cluster config.
func New(opts Options) (*Client, error) {
	cfg, err := restConfig(opts)
	if err != nil {
		return nil, err
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create clientset: %w", err)
	}
	return &Client{cs: cs}, nil
}

func restConfig(opts Options) (*rest.Config, error) {
	kubeconfig := opts.Kubeconfig
	if kubeconfig == "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}
	if kubeconfig == "" {
		if home, err := os.UserHomeDir(); err == nil {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}
	loading := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}
	overrides := &clientcmd.ConfigOverrides{}
	if opts.Context != "" {
		overrides.CurrentContext = opts.Context
	}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loading, overrides).ClientConfig()
	if err == nil {
		return cfg, nil
	}
	// Fall back to in-cluster when kubeconfig is unavailable.
	inCluster, icErr := rest.InClusterConfig()
	if icErr != nil {
		return nil, fmt.Errorf("kubeconfig (%v) and in-cluster (%v) failed", err, icErr)
	}
	return inCluster, nil
}

// ListIngresses returns Ingress objects for a namespace (or all namespaces when empty).
func (c *Client) ListIngresses(ctx context.Context, namespace string) ([]*networkingv1.Ingress, error) {
	ns := namespace
	if ns == "" {
		ns = metav1.NamespaceAll
	}
	list, err := c.cs.NetworkingV1().Ingresses(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list ingresses: %w", err)
	}
	out := make([]*networkingv1.Ingress, 0, len(list.Items))
	for i := range list.Items {
		ing := list.Items[i]
		out = append(out, &ing)
	}
	return out, nil
}
