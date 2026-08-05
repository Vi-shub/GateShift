// Package audit formats migration audit matrices for CLI output.
package audit

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/gateshift/gateshift/pkg/ir"
)

// WriteMatrix prints a human-readable L1/L2/L3 audit matrix.
func WriteMatrix(w io.Writer, bundle *ir.MigrationBundle) {
	direct, policy, blocked := bundle.Summary()
	fmt.Fprintf(w, "GateShift Audit Matrix\n")
	fmt.Fprintf(w, "======================\n")
	if bundle.SchemaVersion != "" {
		fmt.Fprintf(w, "IR: %s\n", bundle.SchemaVersion)
	}
	if len(bundle.RequiredFeatures) > 0 {
		feats := make([]string, len(bundle.RequiredFeatures))
		for i, f := range bundle.RequiredFeatures {
			feats[i] = string(f)
		}
		fmt.Fprintf(w, "Required features: %s\n", strings.Join(feats, ", "))
	}
	fmt.Fprintf(w, "Readiness: %d/100 (%s)\n", bundle.ReadinessScore(), bundle.ReadinessLabel())
	fmt.Fprintf(w, "Summary: %d L1 direct | %d L2 policy | %d L3 manual\n\n", direct, policy, blocked)

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "LVL\tSTATUS\tID\tINGRESS\tANNOTATION / FEATURE\tFIX\tTARGET\tNOTES")
	fmt.Fprintln(tw, "---\t------\t--\t-------\t--------------------\t---\t------\t-----")
	for _, f := range bundle.Findings {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			levelLabel(f.Level, f.Status),
			statusLabel(f.Status),
			display(f.ID),
			display(f.IngressName),
			feature(f),
			display(f.Fix),
			display(f.Target),
			sanitize(f.Message),
		)
	}
	_ = tw.Flush()
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Legend:")
	fmt.Fprintln(w, "  L1 [OK]   Direct Gateway API HTTPRoute filter mapping")
	fmt.Fprintln(w, "  L2 [POL]  Requires provider Policy / Certificate CRD (or unknown annotation)")
	fmt.Fprintln(w, "  L3 [MAN]  Untranslatable nginx magic — manual rewrite required")
	fmt.Fprintln(w, "  FIX       CLI flag or playbook that remediates the finding")
}

func levelLabel(level int, status ir.Status) string {
	if level > 0 {
		return fmt.Sprintf("L%d", level)
	}
	switch status {
	case ir.StatusDirect:
		return "L1"
	case ir.StatusRequiresPolicy:
		return "L2"
	default:
		return "L3"
	}
}

func statusLabel(s ir.Status) string {
	switch s {
	case ir.StatusDirect:
		return "[OK]"
	case ir.StatusRequiresPolicy:
		return "[POL]"
	case ir.StatusUntranslatable:
		return "[MAN]"
	default:
		return strings.ToUpper(string(s))
	}
}

func feature(f ir.AuditFinding) string {
	if f.Value == "" {
		return f.Key
	}
	return fmt.Sprintf("%s=%s", f.Key, truncate(f.Value, 40))
}

func display(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func sanitize(s string) string {
	return strings.ReplaceAll(s, "\t", " ")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
