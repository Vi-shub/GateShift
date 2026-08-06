package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gateshift/gateshift/pkg/convert"
	"github.com/gateshift/gateshift/pkg/ir"
	"github.com/gateshift/gateshift/pkg/loader"
)

func newDualRunCmd() *cobra.Command {
	var (
		file               string
		output             string
		target             string
		gwClass            string
		gwName             string
		noGW               bool
		preserveRegex      bool
		trailingSlashRedir bool
	)
	cmd := &cobra.Command{
		Use:   "dual-run",
		Short: "Emit staging Gateway + shadow HTTPRoute while keeping Ingress live",
		Long: `Dual-run (shadow) cutover helper.

Converts Ingress → Gateway API for parallel validation without deleting or
modifying the source Ingress. Emits a staging Gateway (unless --no-gateway)
and *-shadow HTTPRoutes annotated with gateshift.io/mode=dual-run.

Prints a cutover checklist to stderr; YAML goes to stdout or -o.`,
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

			convertOpts := convert.Options{
				Provider:                   provider,
				GatewayClass:               gwClass,
				GatewayName:                gwName,
				IncludeGateway:             !noGW,
				PreserveNGINXRegex:         preserveRegex,
				EmitTrailingSlashRedirects: trailingSlashRedir,
			}
			// Prefer a staging default name when the user did not override.
			if convertOpts.GatewayName == "" && !noGW {
				if len(ingresses) == 1 {
					convertOpts.GatewayName = ingresses[0].Name + "-staging-gateway"
				} else {
					convertOpts.GatewayName = "shared-staging-gateway"
				}
			}

			bundle, err := convert.FromIngresses(ingresses, convertOpts)
			if err != nil {
				return exitErr(err)
			}
			convert.ApplyDualRunMode(bundle, convert.DualRunOptions{
				GatewayName:    convertOpts.GatewayName,
				IncludeGateway: !noGW,
			})

			yamlBytes, err := convert.EmitYAML(bundle)
			if err != nil {
				return exitErr(err)
			}

			fmt.Fprint(cmd.ErrOrStderr(), convert.FormatDualRunChecklist(bundle))
			fmt.Fprintln(cmd.ErrOrStderr())

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
	cmd.Flags().StringVar(&target, "target", "envoy-gateway", "Target provider: standard|envoy-gateway|cilium|istio|kong")
	cmd.Flags().StringVar(&gwClass, "gateway-class", "envoy", "GatewayClass name for the staging Gateway")
	cmd.Flags().StringVar(&gwName, "gateway-name", "", "Staging Gateway name (default: <ingress>-staging-gateway)")
	cmd.Flags().BoolVar(&noGW, "no-gateway", false, "Emit shadow HTTPRoute/policies only (attach to an existing Gateway)")
	cmd.Flags().BoolVar(&preserveRegex, "preserve-nginx-regex", false, "Emit case-insensitive prefix RegularExpression matches for Ingress-NGINX regex-forced hosts")
	cmd.Flags().BoolVar(&trailingSlashRedir, "emit-trailing-slash-redirects", false, "Emit 301 redirects for /path → /path/ (Ingress-NGINX trailing-slash behavior)")
	return cmd
}
