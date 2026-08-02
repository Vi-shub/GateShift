package patterns

import (
	"testing"

	"github.com/gateshift/gateshift/pkg/ir"
)

func TestAnalyzeSnippetPromotesHeaders(t *testing.T) {
	snippet := `more_set_headers "X-Custom-Header: True";`
	res := AnalyzeSnippet(snippet, ir.ProviderEnvoyGateway, "demo", "default")
	if !res.FullyCovered {
		t.Fatalf("expected full coverage, unmatched=%v hints=%v", res.UnmatchedLines, res.Hints)
	}
	if len(res.Filters) != 1 || res.Filters[0].SetHeaders["X-Custom-Header"] != "True" {
		t.Fatalf("unexpected filters: %#v", res.Filters)
	}
}

func TestAnalyzeSnippetPartialWithUA(t *testing.T) {
	snippet := `
more_set_headers "X-Custom: True";
if ($http_user_agent ~* "bad-bot") { return 403; }
`
	res := AnalyzeSnippet(snippet, ir.ProviderEnvoyGateway, "demo", "default")
	if len(res.Filters) == 0 {
		t.Fatal("expected header filter promotion")
	}
	if len(res.Policies) == 0 {
		t.Fatal("expected UA deny policy scaffold")
	}
}

func TestAnalyzeSnippetLuaStaysUncovered(t *testing.T) {
	snippet := `access_by_lua_block { ngx.exit(403) }`
	res := AnalyzeSnippet(snippet, ir.ProviderStandard, "x", "ns")
	if res.FullyCovered {
		t.Fatal("lua must not be fully covered")
	}
}
