# Roadmap

GateShift’s north star: migrate large Ingress fleets to Gateway API with **explicit annotation reporting** and **without inventing unsafe proxy config**.

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

- [x] goreleaser + GitHub Releases (linux/darwin/windows + arm64)
- [x] CI: `go test`, `gofmt`, corpus scoreboard, GoReleaser check
- [x] KinD smoke workflow (Envoy Gateway) on PR / main
- [x] One-line install script (`scripts/install.sh`)
- [x] Public corpus + multi-provider `gateshift scoreboard` (Envoy / Cilium / Istio / Kong / standard)
- [x] Compare doc + committed scoreboard snapshot
- [ ] Signed containers for `gateshift-operator`
- [ ] Helm chart values docs + install NOTES
- [ ] `--http-only` / TLS secret awareness in convert for lab clusters
- [x] Close tracked NGINX catalog gaps (access-log, custom-http-errors, ssl-passthrough, basic-auth, proxy buffering/retries/timeouts, default-backend)

### P1 — Fleet coverage

- [x] Import public + GitHub Ingress corpus under `examples/corpus/`
- [x] Corpus CI: scoreboard on every PR (unreported annotations must stay 0)
- [ ] Grow corpus toward 50–100 fixtures (charts + more upstream examples)
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
| Unreported annotations | Zero — every annotation produces a finding |
| KinD smoke | Green on release tags |
