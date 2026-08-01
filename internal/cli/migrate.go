package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/gateshift/gateshift/pkg/convert"
	"github.com/gateshift/gateshift/pkg/gitops"
	"github.com/gateshift/gateshift/pkg/ir"
	"github.com/gateshift/gateshift/pkg/loader"
)

func newMigrateCmd() *cobra.Command {
	var (
		file         string
		target       string
		gwClass      string
		repo         string
		baseBranch   string
		dryRunDir    string
		autoPR       bool
		output       string
	)
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Convert Ingress and create a GitOps PR (or dry-run artifacts)",
		Long:  "End-to-end migration helper: convert → audit markdown → open GitHub PR when GITHUB_TOKEN is set, otherwise write local dry-run files.",
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
			name := "migration"
			for _, ing := range ingresses {
				name = ing.Name
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
				combined.Gateways = append(combined.Gateways, bundle.Gateways...)
				combined.Policies = append(combined.Policies, bundle.Policies...)
				combined.Certificates = append(combined.Certificates, bundle.Certificates...)
			}
			yamlBytes, err := convert.EmitYAML(combined)
			if err != nil {
				return exitErr(err)
			}
			if output != "" && output != "-" {
				if err := os.WriteFile(output, yamlBytes, 0o644); err != nil {
					return exitErr(err)
				}
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
			defer cancel()

			dir := dryRunDir
			if !autoPR {
				if dir == "" {
					dir = ".gateshift-pr/" + name
				}
			}
			res, err := gitops.CreateMigrationPR(ctx, gitops.PRRequest{
				Repo:         repo,
				BaseBranch:   baseBranch,
				ManifestYAML: yamlBytes,
				Bundle:       combined,
				IngressName:  name,
				DryRunDir:    dir,
			})
			if err != nil {
				return exitErr(err)
			}
			if res.DryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "Dry-run PR artifacts written:\n")
				for _, f := range res.Files {
					fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", f)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Set GITHUB_TOKEN and --repo owner/name --auto-pr to open a real PR.\n")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Opened PR: %s\n", res.URL)
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "Path to Ingress YAML file")
	cmd.Flags().StringVar(&target, "target", "envoy-gateway", "Target provider")
	cmd.Flags().StringVar(&gwClass, "gateway-class", "envoy", "GatewayClass name")
	cmd.Flags().StringVar(&repo, "repo", "", "GitHub repo owner/name")
	cmd.Flags().StringVar(&baseBranch, "base", "main", "Base branch for PRs")
	cmd.Flags().StringVar(&dryRunDir, "dry-run-dir", "", "Write PR artifacts locally")
	cmd.Flags().BoolVar(&autoPR, "auto-pr", false, "Create a real GitHub PR when GITHUB_TOKEN is set")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Also write converted YAML to this path")
	return cmd
}
