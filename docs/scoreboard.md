# How to read the corpus scoreboard

The scoreboard measures annotation fidelity across a public Ingress corpus for each target provider.

Generate or refresh the committed snapshot:

```bash
make scoreboard
# writes docs/scoreboard.md
```

Or:

```bash
gateshift scoreboard -f examples/corpus -o docs/scoreboard.md
```

Latest numbers: [scoreboard.md](scoreboard.md) · related tools: [COMPARE.md](COMPARE.md) · gaps: [CORPUS_GAPS.md](CORPUS_GAPS.md)

## Columns

| Column | Meaning |
|--------|---------|
| Readiness 0-100 | Migration safety label (`READY`, `READY_WITH_POLICIES`, `NEEDS_REVIEW`, `BLOCKED`) |
| L1 / L2 / L3 | Direct filters · provider Policies · manual / snippets |
| Validate | Controller capability gate for that `--target` |
| Unreported | Must stay **0**. Every migration annotation becomes a finding |
| Structure-only baseline | Annotation keys a hosts/paths/TLS-only conversion would omit |

## How to interpret results

- **Unreported = 0** is a hard quality bar. Unknown or hard features stay as findings; they are not omitted.
- **Validate FAIL** on snippet/WAF fixtures is intentional (fail closed).
- **Provider columns** differ when Policy emission or the capability matrix diverges.
- Envoy Gateway usually has the highest validate pass rate today because L2 Policy emission aligns with its profile. Cilium is close. Istio, Kong, and `standard` fail closed where Extended features are unsupported.

## Corpus layout

| Path | Contents |
|------|----------|
| `examples/corpus/public/` | Curated NGINX / cert-manager scenarios |
| `examples/corpus/github/` | Upstream ingress-nginx style fixtures |
| `examples/corpus/traefik/` (+ edge) | Traefik annotation samples |
| `examples/corpus/community/` | Community migration samples |
| `examples/corpus/blog-k8s-2026-02/` | Ingress-NGINX behavioral quirks |

See [examples/corpus/README.md](../examples/corpus/README.md).
