# Roadmap

GateShift’s north star: migrate large Ingress fleets to Gateway API **without silent annotation loss** and **without inventing unsafe proxy config**.

## Done (current baseline)

- Plug-in L1/L2/L3 annotation adapter engine
- CLI: `audit`, `convert`, `diff`, `validate`, `migrate`, `coverage`
- Snippet pattern library (promote safe idioms)
- Canary Ingress merge → weighted `HTTPRoute`
- Readiness score + annotation catalog
- Provider-oriented Policy emission (Envoy Gateway focus)
- KinD smoke test path
- `MigrationRequest` CRD, operator scaffold, Helm chart stub
- GitOps dry-run / GitHub PR helper

## Next (highest leverage)

### P0 — Production hardening

- [ ] goreleaser + GitHub Releases (linux/darwin/windows)
- [ ] CI: `go test`, `gofmt`, golangci-lint, smoke job on KinD
- [ ] Signed containers for `gateshift-operator`
- [ ] Helm chart values docs + install NOTES
- [ ] `--http-only` / TLS secret awareness in convert for lab clusters
- [ ] Close catalog gaps flagged by `gateshift coverage` (access-log, custom-http-errors, ssl-passthrough, basic-auth, …)

### P1 — Fleet coverage

- [ ] Import public + private Ingress corpus under `examples/corpus/`
- [ ] Corpus CI: every fixture must `audit` and either `validate` or assert expected L3
- [ ] Traefik / AWS ALB / GCE Ingress annotation adapters
- [ ] Live `GatewayClass` feature discovery (replace static matrix where possible)
- [ ] Dual-run helper: keep Ingress + attach HTTPRoute for shadowed traffic

### P2 — Operator maturity

- [ ] Status conditions + events for `MigrationRequest`
- [ ] Prometheus metrics (conversions, L3 counts, PR failures)
- [ ] Admission webhook / policy: block apply when readiness &lt; threshold
- [ ] GitLab + Azure DevOps GitOps backends
- [ ] ReferenceGrant / cross-namespace parentRefs support

### P3 — Ecosystem

- [ ] Homebrew / scoop formulas
- [ ] CNCF / Gateway API community announcement package
- [ ] Conformance report HTML export for change-review boards
- [ ] Optional embedding search over corpus (“nearest known migration”)

## Non-goals

- Full nginx/Lua → Envoy compiler
- Guaranteeing 100% automatic conversion of every Ingress
- Replacing controller-specific Policy CRDs with invented Core API fields

## Success metrics

| Metric | Target |
|--------|--------|
| Catalog implementation | ≥ 90% of tracked NGINX keys |
| Fleet readiness ≥ 60 | ≥ 80% of scanned Ingresses |
| Silent drops | Zero — every annotation produces a finding |
| KinD smoke | Green on release tags |
