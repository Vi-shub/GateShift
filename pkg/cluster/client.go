// Package cluster loads live Ingress resources from a Kubernetes cluster.
package cluster

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// ListOptions filters Ingress listing.
type ListOptions struct {
	// Namespace restricts the list. Empty means all namespaces.
	Namespace string
	// LabelSelector is a standard Kubernetes label selector (e.g. "app=shop").
	LabelSelector string
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

// NewFromClientset builds a Client around an existing clientset (tests / fakes).
func NewFromClientset(cs kubernetes.Interface) *Client {
	return &Client{cs: cs}
}

func restConfig(opts Options) (*rest.Config, error) {
	overrides := &clientcmd.ConfigOverrides{}
	if opts.Context != "" {
		overrides.CurrentContext = opts.Context
	}

	var lastErr error
	for _, kubeconfig := range candidateKubeconfigs(opts.Kubeconfig) {
		loading := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}
		cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loading, overrides).ClientConfig()
		if err == nil {
			return cfg, nil
		}
		lastErr = err
	}

	// Default loading rules (KUBECONFIG env / ~/.kube/config) when no explicit file worked.
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(), overrides).ClientConfig()
	if err == nil {
		return cfg, nil
	}
	if lastErr == nil {
		lastErr = err
	}

	inCluster, icErr := rest.InClusterConfig()
	if icErr != nil {
		return nil, fmt.Errorf("kubeconfig (%v) and in-cluster (%v) failed", lastErr, icErr)
	}
	return inCluster, nil
}

func candidateKubeconfigs(explicit string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			return
		}
		if _, err := os.Stat(p); err != nil {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	add(explicit)
	if env := os.Getenv("KUBECONFIG"); env != "" {
		for _, p := range filepath.SplitList(env) {
			add(p)
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		add(filepath.Join(home, ".kube", "config"))
	}
	// Common distro paths (k3s / rke2) when ~/.kube/config was never copied.
	add("/etc/rancher/k3s/k3s.yaml")
	add("/etc/rancher/rke2/rke2.yaml")
	return out
}

// ListIngresses returns Ingress objects matching ListOptions.
func (c *Client) ListIngresses(ctx context.Context, opts ListOptions) ([]*networkingv1.Ingress, error) {
	ns := opts.Namespace
	if ns == "" {
		ns = metav1.NamespaceAll
	}
	list, err := c.cs.NetworkingV1().Ingresses(ns).List(ctx, metav1.ListOptions{
		LabelSelector: opts.LabelSelector,
	})
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
