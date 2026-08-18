package audit

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gateshift/gateshift/pkg/ir"
)

func TestWriteFleetSummary(t *testing.T) {
	rows := []FleetRow{
		{Namespace: "shop", Name: "checkout", Score: 80, Label: "Ready", Direct: 3, Policy: 1, Blocked: 0},
		{Namespace: "blog", Name: "www", Score: 40, Label: "Needs work", Direct: 1, Policy: 0, Blocked: 2},
	}
	var buf bytes.Buffer
	WriteFleetSummary(&buf, rows)
	out := buf.String()
	for _, want := range []string{
		"Fleet Summary (2 Ingresses)",
		"blog",
		"www",
		"shop",
		"checkout",
		"Fleet totals: 4 L1 | 1 L2 | 2 L3",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestFleetRowFromBundle(t *testing.T) {
	b := &ir.MigrationBundle{
		Findings: []ir.AuditFinding{
			{Status: ir.StatusDirect, Level: 1},
			{Status: ir.StatusRequiresPolicy, Level: 2},
			{Status: ir.StatusUntranslatable, Level: 3},
		},
	}
	row := FleetRowFromBundle("ns", "ing", b)
	if row.Namespace != "ns" || row.Name != "ing" {
		t.Fatalf("identity: %#v", row)
	}
	if row.Direct != 1 || row.Policy != 1 || row.Blocked != 1 {
		t.Fatalf("counts: %#v", row)
	}
}
