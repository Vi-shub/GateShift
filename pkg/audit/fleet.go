package audit

import (
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/gateshift/gateshift/pkg/ir"
)

// FleetRow is one Ingress's readiness snapshot for fleet batch audits.
type FleetRow struct {
	Namespace string
	Name      string
	Score     int
	Label     string
	Direct    int
	Policy    int
	Blocked   int
}

// FleetRowFromBundle builds a FleetRow from a single-Ingress conversion bundle.
func FleetRowFromBundle(namespace, name string, bundle *ir.MigrationBundle) FleetRow {
	d, p, b := bundle.Summary()
	return FleetRow{
		Namespace: namespace,
		Name:      name,
		Score:     bundle.ReadinessScore(),
		Label:     bundle.ReadinessLabel(),
		Direct:    d,
		Policy:    p,
		Blocked:   b,
	}
}

// WriteFleetSummary prints a per-Ingress readiness table, then overall totals.
func WriteFleetSummary(w io.Writer, rows []FleetRow) {
	if len(rows) == 0 {
		return
	}
	sorted := append([]FleetRow(nil), rows...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Namespace != sorted[j].Namespace {
			return sorted[i].Namespace < sorted[j].Namespace
		}
		return sorted[i].Name < sorted[j].Name
	})

	fmt.Fprintf(w, "GateShift Fleet Summary (%d Ingresses)\n", len(sorted))
	fmt.Fprintf(w, "=====================================\n")
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "NAMESPACE\tINGRESS\tSCORE\tLABEL\tL1\tL2\tL3")
	fmt.Fprintln(tw, "---------\t-------\t-----\t-----\t--\t--\t--")
	var td, tp, tb int
	for _, r := range sorted {
		fmt.Fprintf(tw, "%s\t%s\t%d\t%s\t%d\t%d\t%d\n",
			r.Namespace, r.Name, r.Score, r.Label, r.Direct, r.Policy, r.Blocked)
		td += r.Direct
		tp += r.Policy
		tb += r.Blocked
	}
	_ = tw.Flush()
	fmt.Fprintf(w, "\nFleet totals: %d L1 | %d L2 | %d L3\n\n", td, tp, tb)
}
