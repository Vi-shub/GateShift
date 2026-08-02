# GateShift

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](go.mod)

**GateShift** migrates Kubernetes Ingress resources (including NGINX annotations) to [Gateway API](https://gateway-api.sigs.k8s.io/) with explicit fidelity controls.

Unlike converters that silently drop annotations, GateShift classifies every feature as:

| Level | Meaning | Behavior |
|------:|---------|----------|
| **L1** | Native Gateway API | Emits `HTTPRoute` filters (rewrite, redirect, headers) |
| **L2** | Provider extension | Emits Policy CRDs (`BackendTrafficPolicy`, `SecurityPolicy`, `Certificate`, …) |
| **L3** | Untranslatable | Flags snippets/Lua for humans; blocks unsafe apply via `validate` |

**CLI:** `gateshift` · **Operator:** `gateshift-operator` · **License:** Apache 2.0

---

## Why GateShift

| Capability | Baseline `ingress2gateway` | GateShift |
|------------|----------------------------|-----------|
| Hosts / paths / backends | Yes | Yes |
| Annotation fidelity | Often dropped | L1/L2/L3 matrix + readiness score |
| Snippets | Ignored | Pattern library (promote safe idioms) |
| Canary Ingress pairs | Manual | Weighted HTTPRoute merge |
| Controller fit | Rarely checked | `validate` capability matrix |
| GitOps | Manual | `migrate` PR / dry-run artifacts |
| In-cluster | N/A | `MigrationRequest` operator |

Design deep-dive: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) · comparison notes: [docs/BEATING_INGRESS2GATEWAY.md](docs/BEATING_INGRESS2GATEWAY.md)

---

## Install / build

Requirements: Go 1.22+, optional Docker + KinD for cluster tests.

```bash
git clone https://github.com/gateshift/gateshift.git
cd gateshift
make tidy test build
# Linux binary:   bin/gateshift
# Windows binary: bin/gateshift.exe   (make build on Windows)
```

Cross-compile for WSL / Linux from Windows:

```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o bin/gateshift ./cmd/gateshift
```

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
```

Live cluster:

```bash
gateshift audit --namespace shop --target=envoy-gateway
```

Targets: `standard` · `envoy-gateway` · `cilium` · `istio` · `kong`

---

## CLI reference

| Command | Purpose |
|---------|---------|
| `audit` | L1/L2/L3 matrix + readiness score (file or `--namespace`) |
| `convert` | Emit Gateway / HTTPRoute / Policy YAML |
| `diff` | Structural Ingress vs Gateway API view |
| `validate` | Provider capability / conformance gate |
| `migrate` | Convert + GitHub PR or local dry-run pack |
| `coverage` | Catalog coverage and per-file `[OK]/`[GAP]`/`[??]` |
| `version` | Print CLI version |

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

## Annotation coverage (summary)

**L1:** `rewrite-target`, `ssl-redirect`, `force-ssl-redirect`, permanent/temporal redirects, CORS, `from-to-www-redirect`, `app-root`, `x-forwarded-prefix`

**L2:** rate limits, cert-manager issuers, affinity, IP allow/deny, proxy timeouts/body size, backend TLS, canary merge, mirroring, `use-regex`, auth-url scaffolds (Envoy)

**L3 / pattern-assisted:** `configuration-snippet`, `server-snippet`, `modsecurity-snippet`

Run `gateshift coverage` for the full tracked catalog and gaps.

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
charts/gateshift-operator/ Helm chart
config/crd/                CRD manifests
examples/                  Sample Ingress + MigrationRequest
examples/corpus/           Regression fixtures
scripts/                   KinD e2e / smoke tests
docs/                      Architecture, testing, roadmap
```

---

## Documentation

| Doc | Description |
|-----|-------------|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | Pipeline, adapter model, cutover strategy |
| [docs/TESTING.md](docs/TESTING.md) | Unit, CLI, and KinD smoke testing |
| [docs/BEATING_INGRESS2GATEWAY.md](docs/BEATING_INGRESS2GATEWAY.md) | Differentiation and coverage strategy |
| [docs/ROADMAP.md](docs/ROADMAP.md) | Near-term and longer-term work |
| [CONTRIBUTING.md](CONTRIBUTING.md) | How to extend adapters and patterns |

---

## Status

| Area | Maturity |
|------|----------|
| CLI convert / audit / validate | Usable |
| Pattern library / canary merge | Usable |
| KinD smoke path (Envoy Gateway) | Proven |
| Operator / Helm | Scaffold — production-harden before wide deploy |
| Multi-controller (Traefik, ALB, GCE) | Planned |

This project prioritizes **safe, reviewable migration** over claiming 100% automatic conversion.

---

## License

Apache License 2.0 — see [LICENSE](LICENSE).
