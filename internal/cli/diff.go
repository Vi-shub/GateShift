package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gateshift/gateshift/pkg/convert"
	"github.com/gateshift/gateshift/pkg/diff"
	"github.com/gateshift/gateshift/pkg/ir"
	"github.com/gateshift/gateshift/pkg/loader"
)

func newDiffCmd() *cobra.Command {
	var (
		file    string
		target  string
		gwClass string
		gwName  string
	)
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show structural Ingress vs Gateway API diff",
		Long:  "Prints a side-by-side structural comparison of the source Ingress and translated Gateway/HTTPRoute resources.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return exitErr(fmt.Errorf("--file is required"))
			}
			provider, err := ir.ParseProvider(target)
			if err != nil {
				return exitErr(err)
			}
			ingresses, err := loader.LoadIngressFile(file)
			if err != nil {
				return exitErr(err)
			}
			opts := convert.Options{
				Provider:       provider,
				GatewayClass:   gwClass,
				GatewayName:    gwName,
				IncludeGateway: true,
			}
			for _, ing := range ingresses {
				bundle, err := convert.FromIngress(ing, opts)
				if err != nil {
					return exitErr(err)
				}
				diff.WriteSideBySide(cmd.OutOrStdout(), ing, bundle)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "Path to Ingress YAML file")
	cmd.Flags().StringVar(&target, "target", "standard", "Target provider: standard|envoy-gateway|cilium|istio|kong")
	cmd.Flags().StringVar(&gwClass, "gateway-class", "envoy", "GatewayClass name for generated Gateways")
	cmd.Flags().StringVar(&gwName, "gateway-name", "", "Override generated Gateway name")
	return cmd
}
