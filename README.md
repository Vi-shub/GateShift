<p align="center">
  <img src="Logo/Logo.png" alt="GateShift" width="220"/>
</p>

<h1 align="center">GateShift</h1>

<p align="center">
  <strong>Ingress → Gateway API migration with annotation fidelity you can trust.</strong>
</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-Apache_2.0-blue.svg" alt="License"/></a>
  <a href="go.mod"><img src="https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white" alt="Go"/></a>
  <a href="docs/ARCHITECTURE.md"><img src="https://img.shields.io/badge/docs-architecture-0ea5e9" alt="Architecture"/></a>
  <a href="docs/TESTING.md"><img src="https://img.shields.io/badge/tests-KinD%20smoke-22c55e" alt="Tests"/></a>
</p>

<p align="center">
  <a href="#install">Install</a> ·
  <a href="#quick-start">Quick start</a> ·
  <a href="#why-gateshift">Why GateShift</a> ·
  <a href="#corpus-scoreboard">Scoreboard</a> ·
  <a href="#cli-reference">CLI</a> ·
  <a href="docs/COMPARE.md">Comparison</a> ·
  <a href="CONTRIBUTING.md">Contributing</a>
</p>

---

**GateShift** converts Kubernetes Ingress (including NGINX / cert-manager annotations) into [Gateway API](https://gateway-api.sigs.k8s.io/) manifests — without silently dropping policy.

Every feature is classified and reported:

| Level | Meaning | Behavior |
|------:|---------|----------|
| **L1** | Native Gateway API | Emits `HTTPRoute` filters (rewrite, redirect, headers) |
| **L2** | Provider extension | Emits Policy CRDs (`BackendTrafficPolicy`, `SecurityPolicy`, `Certificate`, …) |
| **L3** | Untranslatable | Flags snippets / Lua for humans; `validate` blocks unsafe apply |

**CLI:** `gateshift` · **Operator:** `gateshift-operator` · **License:** Apache 2.0

---

## Why GateShift

| Capability | GateShift |
|------------|-----------|
| Hosts / paths / backends | Yes |
| Annotation fidelity | L1 / L2 / L3 matrix + readiness score |
| Snippets | Pattern library (promote safe idioms); residual L3 reported |
| Canary Ingress pairs | Weighted `HTTPRoute` merge |
| Controller fit | `validate` capability matrix |
| GitOps | `migrate` PR / dry-run artifacts |
| In-cluster | Optional `MigrationRequest` operator |

Related-tool comparison + multi-provider scoreboard: [docs/COMPARE.md](docs/COMPARE.md) · design: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)

---

## Install

One-line (Linux / macOS, after the first GitHub Release):

```bash
curl -fsSL https://raw.githubusercontent.com/gateshift/gateshift/main/scripts/install.sh | bash
```

Or build from source (Go 1.22+):

```bash
git clone https://github.com/gateshift/gateshift.git
cd gateshift
make tidy test build
# Linux:   bin/gateshift
# Windows: bin/gateshift.exe
```

Cross-compile for WSL / Linux from Windows:

```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o bin/gateshift ./cmd/gateshift
```

Release binaries (linux / darwin / windows × amd64 / arm64) are published via GoReleaser on `v*` tags.

---

## Quick start

```bash
# Audit migratability
gateshift audit -f examples/ingress-checkout.yaml --target=envoy-gateway

# Emit Gateway API manifests
gateshift convert -f examples/ingress-checkout.yaml --target=envoy-gateway -o gateway.yaml

# Fail closed on untranslatable features
gateshift validate -f examples/ingress-checkout.yaml --target=envoy-gateway

# Structural comparison
gateshift diff -f examples/ingress-checkout.yaml

# GitOps dry-run (writes .gateshift-pr/); set GITHUB_TOKEN + --auto-pr for a real PR
gateshift migrate -f examples/ingress-checkout.yaml --target=envoy-gateway

# Annotation catalog / gap analysis
gateshift coverage -f examples/ingress-checkout.yaml

# Multi-provider corpus scoreboard (Envoy, Cilium, Istio, Kong, standard)
gateshift scoreboard -f examples/corpus -o docs/scoreboard.md
```

Live cluster:

```bash
gateshift audit --namespace shop --target=envoy-gateway
```

**Targets:** `standard` · `envoy-gateway` · `cilium` · `istio` · `kong`

**End-to-end demo** (real app on KinD): [examples/demo-podinfo](examples/demo-podinfo) · `bash scripts/demo-podinfo.sh`

---

## Corpus scoreboard

GateShift ships a public Ingress corpus and a provider matrix so you can **prove** annotation fidelity instead of claiming it.

```bash
make scoreboard
# → docs/scoreboard.md
```

| What it measures | Meaning |
|------------------|---------|
| Readiness 0–100 | Migration safety (`READY` / `READY_WITH_POLICIES` / `NEEDS_REVIEW` / `BLOCKED`) |
| L1 / L2 / L3 | Direct filters · provider Policies · manual / snippets |
| Validate | Controller capability gate per target |
| Unreported | Always **0** — every migration annotation becomes a finding |
| Structure-only baseline | Annotation keys a hosts/paths/TLS-only conversion would omit |

Providers scored: **Envoy Gateway**, **Cilium**, **Istio**, **Kong**, and portable **standard**. See [docs/COMPARE.md](docs/COMPARE.md) and the latest [docs/scoreboard.md](docs/scoreboard.md).

---

## CLI reference

| Command | Purpose |
|---------|---------|
| `audit` | L1/L2/L3 matrix + readiness score (file or `--namespace`) |
| `convert` | Emit Gateway / HTTPRoute / Policy YAML |
| `diff` | Structural Ingress vs Gateway API view |
| `validate` | Provider capability / conformance gate |
| `migrate` | Convert + GitHub PR or local dry-run pack |
| `coverage` | Catalog coverage and per-key `[OK]` / `[GAP]` / `[??]` |
| `scoreboard` | Corpus report across Envoy / Cilium / Istio / Kong / standard |
| `version` | Print CLI version |
---

## Annotation coverage

**L1** — `rewrite-target`, `ssl-redirect`, `force-ssl-redirect`, permanent/temporal redirects, CORS, `from-to-www-redirect`, `app-root`, `x-forwarded-prefix`

**L2** — rate limits, cert-manager issuers, affinity / session cookies, IP allow/deny, proxy timeouts & body size, backend TLS, canary merge, mirroring, `use-regex`, auth-url scaffolds (Envoy), and the rest of the tracked catalog

**L3 / pattern-assisted** — `configuration-snippet`, `server-snippet`, `modsecurity-snippet`

Tracked catalog coverage is **100%** of listed keys (`gateshift coverage`). Snippets stay L3 by design: pattern-assisted, never silently dropped.

---

## Cluster smoke test (KinD)

See [docs/TESTING.md](docs/TESTING.md).

```bash
# Ubuntu WSL (requires Linux binary at bin/gateshift)
export PATH=$HOME/bin:$PATH
bash scripts/test-smoke.sh
# Expected: PASS and HTTP body checkout-ok
```

---

## Operator

```bash
kubectl apply -f config/crd/migrationrequest.yaml
# or: helm install gateshift-operator charts/gateshift-operator
kubectl apply -f examples/migrationrequest.yaml
make build-operator
```

The reconciler watches `MigrationRequest`, converts the referenced Ingress, updates status, and optionally opens a GitOps PR.

---

## Repository layout

```
api/v1alpha1/              MigrationRequest API
cmd/gateshift/             CLI entrypoint
cmd/gateshift-operator/    Operator manager
internal/cli/              Cobra commands
internal/controller/       Reconciler
pkg/adapters/              AnnotationAdapter plugin interface
pkg/adapters/nginx/        NGINX / cert-manager adapters + catalog
pkg/patterns/              Snippet pattern library
pkg/ir/                    Intermediate representation
pkg/convert/               Ingress → IR → YAML (+ canary merge)
pkg/conformance/           Provider capability checks
pkg/cluster/               Live Ingress listing
pkg/gitops/                GitHub PR + dry-run artifacts
pkg/scoreboard/            Multi-provider corpus scoring
charts/gateshift-operator/ Helm chart
config/crd/                CRD manifests
examples/                  Sample Ingress + demos
examples/corpus/           Public + GitHub + Traefik + community fixtures
scripts/                   Install, KinD smoke / demo
docs/                      Architecture, compare, scoreboard
Logo/                      Project brand asset
.github/workflows/         CI, KinD smoke, GoReleaser release
```

---

## Documentation

| Doc | Description |
|-----|-------------|
| [docs/COMPARE.md](docs/COMPARE.md) | Related tools + provider matrix |
| [docs/scoreboard.md](docs/scoreboard.md) | Generated corpus scoreboard snapshot |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Pipeline, adapter model, cutover strategy |
| [docs/ANNOTATION_FIDELITY.md](docs/ANNOTATION_FIDELITY.md) | Classification model and coverage loop |
| [docs/TESTING.md](docs/TESTING.md) | Unit, CLI, CI, and KinD smoke testing |
| [docs/ROADMAP.md](docs/ROADMAP.md) | Near-term and longer-term work |
| [CONTRIBUTING.md](CONTRIBUTING.md) | How to extend adapters and patterns |
| [SECURITY.md](SECURITY.md) | Vulnerability reporting |
| [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) | Community standards |

---

## Status

| Area | Maturity |
|------|----------|
| CLI convert / audit / validate | Usable |
| Pattern library / canary merge | Usable |
| KinD smoke path (Envoy Gateway) | Proven |
| podinfo end-to-end demo | Proven |
| Operator / Helm | Scaffold — harden before wide deploy |
| Multi-controller (Traefik, ALB, GCE) | Planned |

GateShift prioritizes **safe, reviewable migration** over claiming fully automatic conversion of every Ingress edge case.

---

## License

Apache License 2.0 — see [LICENSE](LICENSE).
