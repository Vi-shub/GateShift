package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gateshift/gateshift/pkg/ir"
	"github.com/gateshift/gateshift/pkg/scoreboard"
)

func newScoreboardCmd() *cobra.Command {
	var (
		root      string
		outPath   string
		providers []string
	)
	cmd := &cobra.Command{
		Use:   "scoreboard",
		Short: "Score corpus fixtures across Gateway providers",
		Long:  "Scans Ingress YAML under a directory and prints a multi-provider readiness and validate scoreboard (Envoy Gateway, Cilium, Istio, Kong, standard).",
		RunE: func(cmd *cobra.Command, args []string) error {
			if root == "" {
				return exitErr(fmt.Errorf("--dir is required (e.g. examples/corpus)"))
			}
			var provs []ir.Provider
			if len(providers) == 0 {
				provs = scoreboard.DefaultProviders
			} else {
				for _, p := range providers {
					parsed, err := ir.ParseProvider(p)
					if err != nil {
						return exitErr(err)
					}
					provs = append(provs, parsed)
				}
			}
			rep, err := scoreboard.Run(scoreboard.Options{Root: root, Providers: provs})
			if err != nil {
				return exitErr(err)
			}
			md := scoreboard.FormatMarkdown(rep)
			if outPath != "" {
				if err := os.WriteFile(outPath, []byte(md), 0o644); err != nil {
					return exitErr(err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Wrote scoreboard to %s (%d fixtures × %d providers)\n",
					outPath, len(rep.Files), len(provs))
			}
			fmt.Fprint(cmd.OutOrStdout(), md)
			return nil
		},
	}
	cmd.Flags().StringVarP(&root, "dir", "f", "examples/corpus", "Corpus directory to scan")
	cmd.Flags().StringVarP(&outPath, "output", "o", "", "Optional markdown output path")
	cmd.Flags().StringSliceVar(&providers, "providers", nil, "Providers (default: all)")
	return cmd
}
