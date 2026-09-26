package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/clientcmd"
)

func newPreflightCmd() *cobra.Command {
	var (
		kubeconfig string
		kubeCtx    string
		target     string
		fix        bool
	)
	cmd := &cobra.Command{
		Use:   "preflight",
		Short: "Check cluster prerequisites for GateShift operations",
		Long: `Preflight verifies that required CRDs and controllers are installed.

Checks performed:
  - Gateway API CRDs (Gateway, HTTPRoute, GatewayClass)
  - Target-specific CRDs (BackendTrafficPolicy, SecurityPolicy for Envoy Gateway)
  - GatewayClass existence

Use --fix to print installation commands for missing components.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPreflight(cmd, kubeconfig, kubeCtx, target, fix)
		},
	}
	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig")
	cmd.Flags().StringVar(&kubeCtx, "context", "", "Kubeconfig context name")
	cmd.Flags().StringVar(&target, "target", "envoy-gateway", "Target provider: standard|envoy-gateway|cilium|istio|kong")
	cmd.Flags().BoolVar(&fix, "fix", false, "Print installation commands for missing components")
	return cmd
}

type preflightResult struct {
	Name    string
	Status  string // "OK", "MISSING", "WARN"
	Message string
	Fix     string
}

func runPreflight(cmd *cobra.Command, kubeconfig, kubeCtx, target string, fix bool) error {
	results := []preflightResult{}

	// Build kubeconfig
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		loadingRules.ExplicitPath = kubeconfig
	}
	overrides := &clientcmd.ConfigOverrides{}
	if kubeCtx != "" {
		overrides.CurrentContext = kubeCtx
	}
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).ClientConfig()
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "❌ Cannot connect to cluster: %v\n", err)
		fmt.Fprintf(cmd.ErrOrStderr(), "\nFix: Set KUBECONFIG or use --kubeconfig flag\n")
		fmt.Fprintf(cmd.ErrOrStderr(), "  export KUBECONFIG=/etc/rancher/k3s/k3s.yaml  # K3s\n")
		fmt.Fprintf(cmd.ErrOrStderr(), "  export KUBECONFIG=~/.kube/config             # Standard\n")
		return exitErr(fmt.Errorf("kubeconfig not found"))
	}

	disc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return exitErr(fmt.Errorf("create discovery client: %w", err))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check Gateway API core CRDs
	gatewayCRDs := []struct {
		group    string
		version  string
		resource string
		name     string
	}{
		{"gateway.networking.k8s.io", "v1", "gateways", "Gateway"},
		{"gateway.networking.k8s.io", "v1", "httproutes", "HTTPRoute"},
		{"gateway.networking.k8s.io", "v1", "gatewayclasses", "GatewayClass"},
	}

	for _, crd := range gatewayCRDs {
		exists := checkCRDExists(ctx, disc, crd.group, crd.version, crd.resource)
		if exists {
			results = append(results, preflightResult{
				Name:   crd.name,
				Status: "OK",
			})
		} else {
			results = append(results, preflightResult{
				Name:    crd.name,
				Status:  "MISSING",
				Message: fmt.Sprintf("CRD %s.%s not found", crd.resource, crd.group),
				Fix:     "kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.0/standard-install.yaml",
			})
		}
	}

	// Check target-specific CRDs
	if target == "envoy-gateway" {
		egCRDs := []struct {
			group    string
			version  string
			resource string
			name     string
		}{
			{"gateway.envoyproxy.io", "v1alpha1", "backendtrafficpolicies", "BackendTrafficPolicy"},
			{"gateway.envoyproxy.io", "v1alpha1", "securitypolicies", "SecurityPolicy"},
		}

		for _, crd := range egCRDs {
			exists := checkCRDExists(ctx, disc, crd.group, crd.version, crd.resource)
			if exists {
				results = append(results, preflightResult{
					Name:   crd.name,
					Status: "OK",
				})
			} else {
				results = append(results, preflightResult{
					Name:    crd.name,
					Status:  "MISSING",
					Message: fmt.Sprintf("Envoy Gateway CRD %s.%s not found", crd.resource, crd.group),
					Fix:     "kubectl apply --server-side -f https://github.com/envoyproxy/gateway/releases/download/v1.2.1/install.yaml",
				})
			}
		}
	}

	// Print results
	fmt.Fprintf(cmd.OutOrStdout(), "\nGateShift Preflight Check (target: %s)\n", target)
	fmt.Fprintf(cmd.OutOrStdout(), "=====================================\n\n")

	allOK := true
	for _, r := range results {
		switch r.Status {
		case "OK":
			fmt.Fprintf(cmd.OutOrStdout(), "✅ %s\n", r.Name)
		case "MISSING":
			fmt.Fprintf(cmd.OutOrStdout(), "❌ %s — %s\n", r.Name, r.Message)
			allOK = false
		case "WARN":
			fmt.Fprintf(cmd.OutOrStdout(), "⚠️  %s — %s\n", r.Name, r.Message)
		}
	}

	if !allOK {
		fmt.Fprintf(cmd.OutOrStdout(), "\n")
		if fix {
			fmt.Fprintf(cmd.OutOrStdout(), "Run these commands to fix:\n\n")
			seen := map[string]bool{}
			for _, r := range results {
				if r.Status == "MISSING" && r.Fix != "" && !seen[r.Fix] {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", r.Fix)
					seen[r.Fix] = true
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\n")
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Run with --fix to see installation commands.\n")
		}
		return exitErr(fmt.Errorf("preflight checks failed"))
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n✅ All preflight checks passed!\n")
	return nil
}

func checkCRDExists(ctx context.Context, disc discovery.DiscoveryInterface, group, version, resource string) bool {
	gv := group + "/" + version
	resources, err := disc.ServerResourcesForGroupVersion(gv)
	if err != nil {
		return false
	}
	for _, r := range resources.APIResources {
		if r.Name == resource {
			return true
		}
	}
	return false
}

// CheckBackendTrafficPolicyCRD checks if BackendTrafficPolicy CRD exists (for skip-extension-policies auto-detect).
func CheckBackendTrafficPolicyCRD(kubeconfig, kubeCtx string) bool {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		loadingRules.ExplicitPath = kubeconfig
	}
	overrides := &clientcmd.ConfigOverrides{}
	if kubeCtx != "" {
		overrides.CurrentContext = kubeCtx
	}
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).ClientConfig()
	if err != nil {
		return false
	}
	disc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return checkCRDExists(ctx, disc, "gateway.envoyproxy.io", "v1alpha1", "backendtrafficpolicies")
}
