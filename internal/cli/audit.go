package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	networkingv1 "k8s.io/api/networking/v1"

	"github.com/gateshift/gateshift/pkg/audit"
	"github.com/gateshift/gateshift/pkg/cluster"
	"github.com/gateshift/gateshift/pkg/convert"
	"github.com/gateshift/gateshift/pkg/ir"
	"github.com/gateshift/gateshift/pkg/loader"
)

func newAuditCmd() *cobra.Command {
	var (
		file          string
		namespace     string
		allNamespaces bool
		selector      string
		target        string
		gwClass       string
		gwName        string
		kubeconfig    string
		kubeCtx       string
	)
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Audit Ingress manifests for Gateway API migratability",
		Long:  "Scans local Ingress YAML or live cluster Ingresses and prints an L1/L2/L3 audit matrix, including Ingress-NGINX behavioral quirk warnings (regex host-wide side effects, trailing-slash redirects, URL normalization). Use --all-namespaces / --selector for fleet batch audits.",
		RunE: func(cmd *cobra.Command, args []string) error {
			provider, err := ir.ParseProvider(target)
			if err != nil {
				return exitErr(err)
			}
			opts := convert.Options{
				Provider:       provider,
				GatewayClass:   gwClass,
				GatewayName:    gwName,
				IncludeGateway: true,
			}

			ingresses, err := loadIngressSources(cmd, ingressSourceOpts{
				File:          file,
				Namespace:     namespace,
				AllNamespaces: allNamespaces,
				Selector:      selector,
				Kubeconfig:    kubeconfig,
				Context:       kubeCtx,
				RequireSource: true,
			})
			if err != nil {
				return exitErr(err)
			}

			rows := make([]audit.FleetRow, 0, len(ingresses))
			for _, ing := range ingresses {
				b, err := convert.FromIngress(ing, opts)
				if err != nil {
					return exitErr(err)
				}
				ns := ing.Namespace
				if ns == "" {
					ns = "default"
				}
				rows = append(rows, audit.FleetRowFromBundle(ns, ing.Name, b))
			}
			if len(rows) > 1 {
				audit.WriteFleetSummary(cmd.OutOrStdout(), rows)
			}

			combined, err := convert.FromIngresses(ingresses, opts)
			if err != nil {
				return exitErr(err)
			}
			audit.WriteMatrix(cmd.OutOrStdout(), combined)
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "Path to Ingress YAML file")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Audit live Ingresses in this namespace")
	cmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "Audit live Ingresses across all namespaces")
	cmd.Flags().StringVarP(&selector, "selector", "l", "", "Label selector for live cluster Ingresses (e.g. team=checkout)")
	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig")
	cmd.Flags().StringVar(&kubeCtx, "context", "", "Kubeconfig context name")
	cmd.Flags().StringVar(&target, "target", "standard", "Target provider: standard|envoy-gateway|cilium|istio|kong")
	cmd.Flags().StringVar(&gwClass, "gateway-class", "envoy", "GatewayClass name for generated Gateways")
	cmd.Flags().StringVar(&gwName, "gateway-name", "", "Override generated Gateway name")
	return cmd
}

type ingressSourceOpts struct {
	File          string
	Namespace     string
	AllNamespaces bool
	Selector      string
	Kubeconfig    string
	Context       string
	RequireSource bool
}

func loadIngressSources(cmd *cobra.Command, o ingressSourceOpts) ([]*networkingv1.Ingress, error) {
	live := o.Namespace != "" || o.AllNamespaces || o.Selector != ""
	switch {
	case o.File != "" && live:
		return nil, fmt.Errorf("use either --file or live cluster flags, not both")
	case o.File != "":
		return loader.LoadIngressFile(o.File)
	case o.Namespace != "" && o.AllNamespaces:
		return nil, fmt.Errorf("use either --namespace or --all-namespaces, not both")
	case live:
		if o.Namespace == "" && !o.AllNamespaces && o.Selector == "" {
			return nil, fmt.Errorf("provide --namespace, --all-namespaces, or --selector")
		}
		// Selector alone implies all namespaces.
		ns := o.Namespace
		if o.AllNamespaces {
			ns = ""
		}
		if o.Selector != "" && o.Namespace == "" {
			ns = ""
		}
		cl, err := cluster.New(cluster.Options{Kubeconfig: o.Kubeconfig, Context: o.Context})
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
		defer cancel()
		ingresses, err := cl.ListIngresses(ctx, cluster.ListOptions{
			Namespace:     ns,
			LabelSelector: o.Selector,
		})
		if err != nil {
			return nil, err
		}
		if len(ingresses) == 0 {
			scope := "cluster"
			if o.Namespace != "" {
				scope = fmt.Sprintf("namespace %q", o.Namespace)
			}
			if o.Selector != "" {
				return nil, fmt.Errorf("no Ingress resources in %s matching selector %q", scope, o.Selector)
			}
			return nil, fmt.Errorf("no Ingress resources in %s", scope)
		}
		return ingresses, nil
	default:
		if o.RequireSource {
			return nil, fmt.Errorf("provide --file, --namespace, --all-namespaces, and/or --selector")
		}
		return nil, fmt.Errorf("no Ingress source")
	}
}
