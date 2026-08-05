package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gateshift/gateshift/pkg/convert"
	"github.com/gateshift/gateshift/pkg/ir"
	"github.com/gateshift/gateshift/pkg/loader"
)

func newConvertCmd() *cobra.Command {
	var (
		file    string
		output  string
		target  string
		gwClass string
		gwName  string
		noGW    bool
	)
	cmd := &cobra.Command{
		Use:   "convert",
		Short: "Convert Ingress YAML to Gateway API manifests",
		Long:  "Translates Ingress resources into Gateway, HTTPRoute, and provider policy YAML.",
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
				IncludeGateway: !noGW,
			}
			combined, err := convert.FromIngresses(ingresses, opts)
			if err != nil {
				return exitErr(err)
			}
			yamlBytes, err := convert.EmitYAML(combined)
			if err != nil {
				return exitErr(err)
			}
			if output == "" || output == "-" {
				_, err = cmd.OutOrStdout().Write(yamlBytes)
				return err
			}
			if err := os.WriteFile(output, yamlBytes, 0o644); err != nil {
				return exitErr(err)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Wrote %s\n", output)
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "Path to Ingress YAML file")
	cmd.Flags().StringVarP(&output, "output", "o", "-", "Output file path (default: stdout)")
	cmd.Flags().StringVar(&target, "target", "standard", "Target provider: standard|envoy-gateway|cilium|istio|kong")
	cmd.Flags().StringVar(&gwClass, "gateway-class", "envoy", "GatewayClass name for generated Gateways")
	cmd.Flags().StringVar(&gwName, "gateway-name", "", "Override generated Gateway name")
	cmd.Flags().BoolVar(&noGW, "no-gateway", false, "Emit HTTPRoute/policies only (attach to an existing Gateway)")
	return cmd
}
