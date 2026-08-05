package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "0.3.0-dev"
)

// NewRootCommand builds the gateshift CLI root.
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "gateshift",
		Short:         "Migrate Kubernetes Ingress resources to Gateway API",
		Long:          "GateShift converts Kubernetes Ingress resources and annotations into Gateway API manifests using a Level 1/2/3 adapter engine, pattern library, canary merge, readiness scoring, and multi-provider validation.",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.AddCommand(newAuditCmd())
	cmd.AddCommand(newConvertCmd())
	cmd.AddCommand(newDiffCmd())
	cmd.AddCommand(newValidateCmd())
	cmd.AddCommand(newMigrateCmd())
	cmd.AddCommand(newCoverageCmd())
	cmd.AddCommand(newScoreboardCmd())
	cmd.AddCommand(newVersionCmd())
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print GateShift version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(os.Stdout, "gateshift %s\n", version)
		},
	}
}

func exitErr(err error) error {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	return err
}
