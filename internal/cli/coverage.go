package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/gateshift/gateshift/pkg/adapters/nginx"
	"github.com/gateshift/gateshift/pkg/loader"
)

func newCoverageCmd() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "coverage",
		Short: "Show annotation catalog coverage (and optional file gap analysis)",
		Long:  "Reports how much of GateShift's tracked NGINX annotation catalog is implemented, and optionally which keys in a file are still gaps vs ingress2gateway-class converters.",
		RunE: func(cmd *cobra.Command, args []string) error {
			stats := nginx.CatalogCoverage()
			fmt.Fprintf(cmd.OutOrStdout(), "GateShift Annotation Catalog Coverage\n")
			fmt.Fprintf(cmd.OutOrStdout(), "====================================\n")
			fmt.Fprintf(cmd.OutOrStdout(), "Implemented: %d / %d (%.1f%%)\n", stats.Implemented, stats.Total, stats.Percent)
			fmt.Fprintf(cmd.OutOrStdout(), "By level among implemented: L1=%d L2=%d L3=%d\n\n",
				stats.ByLevel[1], stats.ByLevel[2], stats.ByLevel[3])

			if len(stats.MissingKeys) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Tracked gaps (not yet implemented):")
				for _, k := range stats.MissingKeys {
					fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", k)
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}

			if file == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "Tip: pass -f ingress.yaml to analyze keys present in a real manifest.")
				return nil
			}
			ingresses, err := loader.LoadIngressFile(file)
			if err != nil {
				return exitErr(err)
			}
			seen := map[string]string{}
			for _, ing := range ingresses {
				for k, v := range ing.Annotations {
					seen[k] = v
				}
			}
			keys := make([]string, 0, len(seen))
			for k := range seen {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			cat := map[string]nginx.CatalogEntry{}
			for _, e := range nginx.Catalog() {
				cat[e.Key] = e
			}

			fmt.Fprintf(cmd.OutOrStdout(), "File analysis: %s\n", file)
			var impl, gap, unknown int
			for _, k := range keys {
				if e, ok := cat[k]; ok {
					if e.Implemented {
						impl++
						fmt.Fprintf(cmd.OutOrStdout(), "  [OK]  L%d  %s\n", e.Level, k)
					} else {
						gap++
						fmt.Fprintf(cmd.OutOrStdout(), "  [GAP] L%d  %s  (%s)\n", e.Level, k, e.Notes)
					}
					continue
				}
				// Non-migration noise (kubectl last-applied, etc.)
				if isNoiseAnnotation(k) {
					continue
				}
				unknown++
				fmt.Fprintf(cmd.OutOrStdout(), "  [??]  ???  %s  (unknown — candidate for new adapter/pattern)\n", k)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\nFile keys: %d implemented | %d catalog gaps | %d unknown\n", impl, gap, unknown)
			return nil
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "Optional Ingress YAML to analyze")
	return cmd
}

func isNoiseAnnotation(k string) bool {
	switch {
	case k == "kubectl.kubernetes.io/last-applied-configuration":
		return true
	case len(k) > 16 && k[:16] == "kubernetes.io/":
		return true
	default:
		return false
	}
}
