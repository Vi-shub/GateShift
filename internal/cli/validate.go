package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gateshift/gateshift/pkg/conformance"
	"github.com/gateshift/gateshift/pkg/convert"
	"github.com/gateshift/gateshift/pkg/ir"
	"github.com/gateshift/gateshift/pkg/loader"
)

func newValidateCmd() *cobra.Command {
	var (
		file    string
		target  string
		gwClass string
	)
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate converted features against a target Gateway controller",
		Long:  "Converts Ingress YAML and checks emitted Gateway API features against the provider capability matrix (Core vs Extended).",
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
			combined := &ir.MigrationBundle{}
			for _, ing := range ingresses {
				bundle, err := convert.FromIngress(ing, convert.Options{
					Provider:       provider,
					GatewayClass:   gwClass,
					IncludeGateway: true,
				})
				if err != nil {
					return exitErr(err)
				}
				combined.Findings = append(combined.Findings, bundle.Findings...)
				combined.HTTPRoutes = append(combined.HTTPRoutes, bundle.HTTPRoutes...)
				combined.Policies = append(combined.Policies, bundle.Policies...)
				combined.Gateways = append(combined.Gateways, bundle.Gateways...)
			}
			result := conformance.ValidateBundle(combined, provider)
			fmt.Fprint(cmd.OutOrStdout(), conformance.Format(result))
			if !result.OK {
				return exitErr(fmt.Errorf("conformance validation failed"))
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "Path to Ingress YAML file")
	cmd.Flags().StringVar(&target, "target", "envoy-gateway", "Target provider")
	cmd.Flags().StringVar(&gwClass, "gateway-class", "envoy", "GatewayClass name")
	return cmd
}
