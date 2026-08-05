// Package scoreboard builds multi-provider corpus reports for annotation
// fidelity, readiness, and controller validate outcomes.
package scoreboard

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/gateshift/gateshift/pkg/conformance"
	"github.com/gateshift/gateshift/pkg/convert"
	"github.com/gateshift/gateshift/pkg/ir"
	"github.com/gateshift/gateshift/pkg/loader"
)

// DefaultProviders is the public scoreboard matrix.
var DefaultProviders = []ir.Provider{
	ir.ProviderStandard,
	ir.ProviderEnvoyGateway,
	ir.ProviderCilium,
	ir.ProviderIstio,
	ir.ProviderKong,
}

// FileResult is one Ingress fixture evaluated for one provider.
type FileResult struct {
	File           string      `json:"file"`
	Provider       ir.Provider `json:"provider"`
	IngressCount   int         `json:"ingressCount"`
	AnnotationKeys int         `json:"annotationKeys"`
	L1             int         `json:"l1"`
	L2             int         `json:"l2"`
	L3             int         `json:"l3"`
	Readiness      int         `json:"readiness"`
	Label          string      `json:"label"`
	ValidateOK     bool        `json:"validateOK"`
	ValidateIssues int         `json:"validateIssues"`
	Unreported     int         `json:"unreported"` // always 0 for GateShift (every key → finding)
	// StructureOnlyBaseline counts annotations a hosts/paths/TLS-only conversion would omit.
	StructureOnlyBaseline int `json:"structureOnlyBaseline"`
}

// ProviderSummary aggregates a provider column.
type ProviderSummary struct {
	Provider                   ir.Provider `json:"provider"`
	Files                      int         `json:"files"`
	ReadyOrPolicies            int         `json:"readyOrPolicies"` // score >= 60
	NeedsReview                int         `json:"needsReview"`
	Blocked                    int         `json:"blocked"`
	ValidatePass               int         `json:"validatePass"`
	ValidateFail               int         `json:"validateFail"`
	TotalL1                    int         `json:"totalL1"`
	TotalL2                    int         `json:"totalL2"`
	TotalL3                    int         `json:"totalL3"`
	TotalUnreported            int         `json:"totalUnreported"`
	TotalStructureOnlyBaseline int         `json:"totalStructureOnlyBaseline"`
	AvgReadiness               float64     `json:"avgReadiness"`
}

// Report is a full corpus scoreboard.
type Report struct {
	Root       string            `json:"root"`
	Files      []string          `json:"files"`
	Results    []FileResult      `json:"results"`
	ByProvider []ProviderSummary `json:"byProvider"`
}

// Options controls corpus discovery.
type Options struct {
	Root      string
	Providers []ir.Provider
}

// Run scans YAML fixtures under root and scores each provider.
func Run(opts Options) (*Report, error) {
	root := opts.Root
	if root == "" {
		return nil, fmt.Errorf("scoreboard root is required")
	}
	providers := opts.Providers
	if len(providers) == 0 {
		providers = append([]ir.Provider{}, DefaultProviders...)
	}

	files, err := discoverYAML(root)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no Ingress YAML fixtures under %s", root)
	}

	var usable []string
	rep := &Report{Root: root}
	for _, file := range files {
		ingresses, err := loader.LoadIngressFile(file)
		if err != nil {
			// Skip non-Ingress YAML (Services, Deployments, docs bundles).
			if strings.Contains(err.Error(), "no Ingress resources found") {
				continue
			}
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		usable = append(usable, file)
		annKeys := countMigrationAnnotations(ingresses)
		rel := relPath(root, file)
		baselineDrops := annKeys // structure-only tools typically ignore these

		for _, provider := range providers {
			bundle, err := convert.FromIngresses(ingresses, convert.Options{
				Provider:       provider,
				GatewayClass:   defaultClass(provider),
				IncludeGateway: true,
			})
			if err != nil {
				return nil, fmt.Errorf("%s (%s): %w", file, provider, err)
			}
			l1, l2, l3 := bundle.Summary()
			vres := conformance.ValidateBundle(bundle, provider)
			rep.Results = append(rep.Results, FileResult{
				File:                  rel,
				Provider:              provider,
				IngressCount:          len(ingresses),
				AnnotationKeys:        annKeys,
				L1:                    l1,
				L2:                    l2,
				L3:                    l3,
				Readiness:             bundle.ReadinessScore(),
				Label:                 bundle.ReadinessLabel(),
				ValidateOK:            vres.OK,
				ValidateIssues:        len(vres.Issues),
				Unreported:            0,
				StructureOnlyBaseline: baselineDrops,
			})
		}
	}
	if len(usable) == 0 {
		return nil, fmt.Errorf("no Ingress YAML fixtures under %s", root)
	}
	rep.Files = usable
	rep.ByProvider = summarize(rep.Results, providers)
	return rep, nil
}

func discoverYAML(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		name := strings.ToLower(d.Name())
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			return nil
		}
		if strings.EqualFold(name, "readme.md") {
			return nil
		}
		// Skip non-Ingress docs accidentally named yaml in corpus (none today).
		out = append(out, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func countMigrationAnnotations(ingresses []*networkingv1.Ingress) int {
	seen := map[string]struct{}{}
	for _, ing := range ingresses {
		if ing == nil {
			continue
		}
		for k := range ing.Annotations {
			if isNoise(k) {
				continue
			}
			seen[k] = struct{}{}
		}
	}
	return len(seen)
}

func isNoise(k string) bool {
	switch {
	case k == "kubectl.kubernetes.io/last-applied-configuration":
		return true
	case strings.HasPrefix(k, "kubernetes.io/"):
		return true
	default:
		return false
	}
}

func defaultClass(p ir.Provider) string {
	switch p {
	case ir.ProviderEnvoyGateway:
		return "envoy"
	case ir.ProviderCilium:
		return "cilium"
	case ir.ProviderIstio:
		return "istio"
	case ir.ProviderKong:
		return "kong"
	default:
		return "envoy"
	}
}

func relPath(root, file string) string {
	rel, err := filepath.Rel(root, file)
	if err != nil {
		return filepath.ToSlash(file)
	}
	return filepath.ToSlash(rel)
}

func summarize(results []FileResult, providers []ir.Provider) []ProviderSummary {
	out := make([]ProviderSummary, 0, len(providers))
	for _, p := range providers {
		var s ProviderSummary
		s.Provider = p
		sumReady := 0
		for _, r := range results {
			if r.Provider != p {
				continue
			}
			s.Files++
			sumReady += r.Readiness
			s.TotalL1 += r.L1
			s.TotalL2 += r.L2
			s.TotalL3 += r.L3
			s.TotalUnreported += r.Unreported
			s.TotalStructureOnlyBaseline += r.StructureOnlyBaseline
			switch {
			case r.Readiness >= 60:
				s.ReadyOrPolicies++
			case r.Readiness >= 35:
				s.NeedsReview++
			default:
				s.Blocked++
			}
			if r.ValidateOK {
				s.ValidatePass++
			} else {
				s.ValidateFail++
			}
		}
		if s.Files > 0 {
			s.AvgReadiness = float64(sumReady) / float64(s.Files)
		}
		out = append(out, s)
	}
	return out
}

// FormatMarkdown renders a GitHub-friendly scoreboard.
func FormatMarkdown(rep *Report) string {
	var b strings.Builder
	b.WriteString("# GateShift corpus scoreboard\n\n")
	b.WriteString("Generated by `gateshift scoreboard`. GateShift reports every migration annotation as an L1/L2/L3 finding ")
	b.WriteString("(**unreported = 0**). The structure-only baseline counts annotation keys that a hosts/paths/TLS-only conversion would not represent.\n\n")

	b.WriteString("## Provider summary\n\n")
	b.WriteString("| Provider | Fixtures | Avg readiness | ≥60 ready | Needs review | Blocked | Validate pass | Unreported | Structure-only baseline |\n")
	b.WriteString("|----------|---------:|--------------:|----------:|-------------:|--------:|--------------:|-----------:|------------------------:|\n")
	for _, s := range rep.ByProvider {
		fmt.Fprintf(&b, "| `%s` | %d | %.1f | %d | %d | %d | %d/%d | **%d** | %d |\n",
			s.Provider, s.Files, s.AvgReadiness, s.ReadyOrPolicies, s.NeedsReview, s.Blocked,
			s.ValidatePass, s.Files, s.TotalUnreported, s.TotalStructureOnlyBaseline)
	}

	b.WriteString("\n## Per-fixture detail (Envoy Gateway)\n\n")
	b.WriteString("| Fixture | Ann keys | L1 | L2 | L3 | Readiness | Validate | Structure-only baseline |\n")
	b.WriteString("|---------|---------:|---:|---:|---:|----------:|:--------:|------------------------:|\n")
	for _, r := range rep.Results {
		if r.Provider != ir.ProviderEnvoyGateway {
			continue
		}
		v := "PASS"
		if !r.ValidateOK {
			v = "FAIL"
		}
		fmt.Fprintf(&b, "| `%s` | %d | %d | %d | %d | %d (%s) | %s | %d |\n",
			r.File, r.AnnotationKeys, r.L1, r.L2, r.L3, r.Readiness, r.Label, v, r.StructureOnlyBaseline)
	}

	b.WriteString("\n## Multi-provider readiness matrix\n\n")
	b.WriteString("| Fixture | standard | envoy-gateway | cilium | istio | kong |\n")
	b.WriteString("|---------|---------:|--------------:|-------:|------:|-----:|\n")

	byFile := map[string]map[ir.Provider]FileResult{}
	var order []string
	for _, r := range rep.Results {
		if _, ok := byFile[r.File]; !ok {
			byFile[r.File] = map[ir.Provider]FileResult{}
			order = append(order, r.File)
		}
		byFile[r.File][r.Provider] = r
	}
	for _, file := range order {
		row := byFile[file]
		fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s | %s |\n",
			file,
			cell(row[ir.ProviderStandard]),
			cell(row[ir.ProviderEnvoyGateway]),
			cell(row[ir.ProviderCilium]),
			cell(row[ir.ProviderIstio]),
			cell(row[ir.ProviderKong]),
		)
	}

	b.WriteString("\n## How to read this\n\n")
	b.WriteString("- **Unreported = 0** by design: unknown or hard features remain L3 findings.\n")
	b.WriteString("- **Validate FAIL** on L3-heavy fixtures is expected (fail closed).\n")
	b.WriteString("- **Provider columns** differ when Policy emission or the capability matrix diverges.\n")
	b.WriteString("- Re-run: `gateshift scoreboard -f examples/corpus -o docs/scoreboard.md`\n")
	return b.String()
}

func cell(r FileResult) string {
	if r.File == "" {
		return "-"
	}
	mark := "✓"
	if !r.ValidateOK {
		mark = "✗"
	}
	return fmt.Sprintf("%d %s", r.Readiness, mark)
}
