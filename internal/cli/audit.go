package cli

import (
	"context"
	"fmt"
	"time"

	networkingv1 "k8s.io/api/networking/v1"
	"github.com/spf13/cobra"

	"github.com/gateshift/gateshift/pkg/audit"
	"github.com/gateshift/gateshift/pkg/cluster"
	"github.com/gateshift/gateshift/pkg/convert"
	"github.com/gateshift/gateshift/pkg/ir"
	"github.com/gateshift/gateshift/pkg/loader"
)

func newAuditCmd() *cobra.Command {
	var (
		file       string
		namespace  string
		target     string
		gwClass    string
		gwName     string
		kubeconfig string
		kubeCtx    string
	)
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Audit Ingress manifests for Gateway API migratability",
		Long:  "Scans local Ingress YAML or live cluster Ingresses and prints an L1/L2/L3 audit matrix.",
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

			var ingresses []*networkingv1.Ingress
			switch {
			case file != "":
				ingresses, err = loader.LoadIngressFile(file)
				if err != nil {
					return exitErr(err)
				}
			case namespace != "":
				cl, err := cluster.New(cluster.Options{Kubeconfig: kubeconfig, Context: kubeCtx})
				if err != nil {
					return exitErr(err)
				}
				ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
				defer cancel()
				ingresses, err = cl.ListIngresses(ctx, namespace)
				if err != nil {
					return exitErr(err)
				}
				if len(ingresses) == 0 {
					return exitErr(fmt.Errorf("no Ingress resources in namespace %q", namespace))
				}
			default:
				return exitErr(fmt.Errorf("provide --file or --namespace (live cluster audit)"))
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
	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig")
	cmd.Flags().StringVar(&kubeCtx, "context", "", "Kubeconfig context name")
	cmd.Flags().StringVar(&target, "target", "standard", "Target provider: standard|envoy-gateway|cilium|istio|kong")
	cmd.Flags().StringVar(&gwClass, "gateway-class", "envoy", "GatewayClass name for generated Gateways")
	cmd.Flags().StringVar(&gwName, "gateway-name", "", "Override generated Gateway name")
	return cmd
}
