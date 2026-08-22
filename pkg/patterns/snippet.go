// Package patterns is GateShift's "learning" layer: a curated library of real-world
// nginx snippet / annotation idioms observed across public Ingress corpora.
//
// This is deliberately not ML. Migration reliability comes from ranked, tested
// pattern matchers that promote L3 snippets into L1 filters or L2 policies when
// the entire snippet is accounted for — and leave residual complexity as L3.
package patterns

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gateshift/gateshift/pkg/ir"
)

// Match is one recognized idiom inside a snippet.
type Match struct {
	ID          string
	Description string
	Confidence  float64 // 0..1
	Filters     []ir.FilterIR
	Policies    []ir.PolicyIR
	Residual    bool // true if this match does not fully explain the snippet
}

// Result summarizes pattern application for a snippet body.
type Result struct {
	Matches        []Match
	Filters        []ir.FilterIR
	Policies       []ir.PolicyIR
	FullyCovered   bool
	CoverageRatio  float64
	UnmatchedLines []string
	Hints          []string
}

var (
	reMoreSetHeaders = regexp.MustCompile(`(?i)more_set_headers\s+"([^:]+):\s*([^"]+)"\s*;`)
	reAddHeader      = regexp.MustCompile(`(?i)add_header\s+([A-Za-z0-9_-]+)\s+"([^"]+)"\s+always\s*;`)
	reAddHeader2     = regexp.MustCompile(`(?i)add_header\s+([A-Za-z0-9_-]+)\s+"([^"]+)"\s*;`)
	reReturnOnly     = regexp.MustCompile(`(?i)^\s*return\s+(\d+)(?:\s+"([^"]*)")?;\s*$`)
	reReturnAny      = regexp.MustCompile(`(?i)return\s+(\d+)`)
	reIfUADeny       = regexp.MustCompile(`(?is)if\s*\(\s*\$http_user_agent\s*~\*?\s*"([^"]+)"\s*\)\s*\{\s*return\s+403\s*;\s*\}`)
	reProxyHide      = regexp.MustCompile(`(?i)proxy_hide_header\s+([A-Za-z0-9_-]+)\s*;`)
	reRewriteBreak   = regexp.MustCompile(`(?i)rewrite\s+\^(.+?)\s+(\S+)\s+break\s*;`)
	reLua            = regexp.MustCompile(`(?i)(access_by_lua|content_by_lua|lua_|balancer_by_lua)`)
	reIfBlock        = regexp.MustCompile(`(?is)if\s*\([^)]+\)\s*\{`)
)

// AnalyzeSnippet extracts known idioms and reports whether the snippet is fully covered.
func AnalyzeSnippet(snippet string, provider ir.Provider, metaName, metaNS string) Result {
	raw := strings.TrimSpace(snippet)
	res := Result{}
	if raw == "" {
		res.FullyCovered = true
		res.CoverageRatio = 1
		return res
	}

	working := raw

	// 1) more_set_headers / add_header → ResponseHeaderModifier
	set := map[string]string{}
	for _, re := range []*regexp.Regexp{reMoreSetHeaders, reAddHeader, reAddHeader2} {
		for _, m := range re.FindAllStringSubmatch(working, -1) {
			set[strings.TrimSpace(m[1])] = strings.TrimSpace(m[2])
			working = strings.Replace(working, m[0], "", 1)
		}
	}
	if len(set) > 0 {
		f := ir.FilterIR{Kind: ir.FilterResponseHeader, SetHeaders: set}
		res.Filters = append(res.Filters, f)
		res.Matches = append(res.Matches, Match{
			ID:          "header-set",
			Description: "Static response headers via more_set_headers/add_header",
			Confidence:  0.95,
			Filters:     []ir.FilterIR{f},
		})
		res.Hints = append(res.Hints, "promoted static headers to ResponseHeaderModifier")
	}

	// 2) proxy_hide_header → response header remove
	var remove []string
	for _, m := range reProxyHide.FindAllStringSubmatch(working, -1) {
		remove = append(remove, m[1])
		working = strings.Replace(working, m[0], "", 1)
	}
	if len(remove) > 0 {
		f := ir.FilterIR{Kind: ir.FilterResponseHeader, RemoveHeaders: remove}
		res.Filters = append(res.Filters, f)
		res.Matches = append(res.Matches, Match{
			ID:          "header-remove",
			Description: "proxy_hide_header → ResponseHeaderModifier.remove",
			Confidence:  0.9,
			Filters:     []ir.FilterIR{f},
		})
	}

	// 3) UA deny if-block → residual scaffold (no invalid EG SecurityPolicy)
	if m := reIfUADeny.FindStringSubmatch(working); len(m) == 2 {
		if provider == ir.ProviderEnvoyGateway {
			res.Matches = append(res.Matches, Match{
				ID:          "ua-deny",
				Description: "if ($http_user_agent) return 403 — complete as SecurityPolicy/WASM manually",
				Confidence:  0.75,
				Residual:    true,
			})
			res.Hints = append(res.Hints, fmt.Sprintf("UA deny pattern %q needs a SecurityPolicy/WASM rule (not auto-emitted for Envoy Gateway)", m[1]))
		} else {
			pol := ir.PolicyIR{
				Kind:      ir.PolicyIPFilter,
				Name:      metaName + "-ua-deny",
				Namespace: metaNS,
				Provider:  provider,
				TargetRef: ir.ParentRefIR{Name: metaName, Namespace: metaNS},
				Spec: map[string]any{
					"apiVersion": "gateshift.io/v1alpha1",
					"kind":       "UserAgentDenyPolicy",
					"pattern":    m[1],
					"action":     "Deny",
					"note":       "Scaffold — map to Envoy SecurityPolicy/WASM or controller-specific UA filter",
				},
			}
			res.Policies = append(res.Policies, pol)
			res.Matches = append(res.Matches, Match{
				ID:          "ua-deny",
				Description: "if ($http_user_agent) return 403",
				Confidence:  0.75,
				Policies:    []ir.PolicyIR{pol},
				Residual:    true,
			})
			res.Hints = append(res.Hints, fmt.Sprintf("UA deny pattern %q promoted to SecurityPolicy scaffold", m[1]))
		}
		working = strings.Replace(working, m[0], "", 1)
	}

	// 4) bare return N → RequestRedirect/DirectResponse hint as filter redirect only for 301/302
	lines := splitMeaningful(working)
	allReturn := len(lines) > 0
	for _, line := range lines {
		if !reReturnOnly.MatchString(line) {
			allReturn = false
			break
		}
	}
	if allReturn {
		m := reReturnOnly.FindStringSubmatch(lines[0])
		code := atoi(m[1])
		if code == 301 || code == 302 {
			scheme := "https"
			f := ir.FilterIR{Kind: ir.FilterRequestRedirect, RedirectScheme: &scheme, RedirectStatusCode: &code}
			res.Filters = append(res.Filters, f)
			res.Matches = append(res.Matches, Match{
				ID:          "return-redirect",
				Description: "return 301/302 promoted to RequestRedirect",
				Confidence:  0.7,
				Filters:     []ir.FilterIR{f},
			})
			working = ""
		}
	}

	// 5) rewrite ... break → URLRewrite hint (partial)
	if m := reRewriteBreak.FindStringSubmatch(working); len(m) == 3 {
		target := m[2]
		f := ir.FilterIR{Kind: ir.FilterURLRewrite, ReplacePrefixMatch: &target}
		res.Filters = append(res.Filters, f)
		res.Matches = append(res.Matches, Match{
			ID:          "rewrite-break",
			Description: "rewrite ... break → URLRewrite (verify regex equivalence)",
			Confidence:  0.55,
			Filters:     []ir.FilterIR{f},
			Residual:    true,
		})
		working = strings.Replace(working, m[0], "", 1)
		res.Hints = append(res.Hints, "rewrite-break promoted with medium confidence — verify path semantics")
	}

	// Residual classification
	if reLua.MatchString(raw) {
		res.Hints = append(res.Hints, "Lua detected — cannot auto-translate; redesign as filter/WASM/extAuth")
	}
	if reIfBlock.MatchString(working) {
		res.Hints = append(res.Hints, "residual if-blocks remain — keep as L3")
	}
	if reReturnAny.MatchString(working) && working != "" {
		res.Hints = append(res.Hints, "residual return directives remain")
	}

	res.UnmatchedLines = splitMeaningful(working)
	covered := 1.0
	if len(splitMeaningful(raw)) > 0 {
		unmatched := float64(len(res.UnmatchedLines))
		total := float64(len(splitMeaningful(raw)))
		covered = 1.0 - (unmatched / total)
		if covered < 0 {
			covered = 0
		}
	}
	res.CoverageRatio = covered
	// Fully covered only when every match is high-confidence and no residual lines/lua remain.
	res.FullyCovered = len(res.UnmatchedLines) == 0 && !reLua.MatchString(raw) && len(res.Matches) > 0
	for _, m := range res.Matches {
		if m.Residual || m.Confidence < 0.8 {
			res.FullyCovered = false
		}
	}
	return res
}

func splitMeaningful(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// drop leftover braces from removed if-blocks
		if line == "{" || line == "}" {
			continue
		}
		out = append(out, line)
	}
	// Also split semicolon-separated single-line snippets
	if len(out) == 1 && strings.Count(out[0], ";") > 1 {
		parts := strings.Split(out[0], ";")
		out = nil
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p+";")
			}
		}
	}
	return out
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
