package scoreboard_test

import (
	"path/filepath"
	"testing"

	"github.com/gateshift/gateshift/pkg/ir"
	"github.com/gateshift/gateshift/pkg/scoreboard"
)

func TestCorpusScoreboardUnreportedZero(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "corpus")
	rep, err := scoreboard.Run(scoreboard.Options{Root: root})
	if err != nil {
		t.Fatalf("scoreboard: %v", err)
	}
	if len(rep.Files) < 10 {
		t.Fatalf("expected at least 10 fixtures, got %d", len(rep.Files))
	}
	if len(rep.ByProvider) != len(scoreboard.DefaultProviders) {
		t.Fatalf("provider summaries: got %d want %d", len(rep.ByProvider), len(scoreboard.DefaultProviders))
	}
	for _, s := range rep.ByProvider {
		if s.TotalUnreported != 0 {
			t.Fatalf("%s unreported = %d, want 0", s.Provider, s.TotalUnreported)
		}
		if s.Files != len(rep.Files) {
			t.Fatalf("%s files=%d want %d", s.Provider, s.Files, len(rep.Files))
		}
	}
	// Envoy should be present and score at least one ready fixture.
	var envoy *scoreboard.ProviderSummary
	for i := range rep.ByProvider {
		if rep.ByProvider[i].Provider == ir.ProviderEnvoyGateway {
			envoy = &rep.ByProvider[i]
			break
		}
	}
	if envoy == nil {
		t.Fatal("missing envoy-gateway summary")
	}
	if envoy.ReadyOrPolicies == 0 {
		t.Fatal("expected some envoy fixtures with readiness >= 60")
	}
	md := scoreboard.FormatMarkdown(rep)
	if len(md) < 200 {
		t.Fatalf("markdown too short: %d", len(md))
	}
}
